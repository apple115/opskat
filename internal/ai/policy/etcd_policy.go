package policy

import (
	"context"
	"path"
	"strings"

	"github.com/cago-frame/cago/pkg/logger"
	"go.uber.org/zap"

	"github.com/opskat/opskat/internal/ai/aictx"
	"github.com/opskat/opskat/internal/model/entity/asset_entity"
)

// ExtractEtcdCommand 提取 etcd 命令名和参数
func ExtractEtcdCommand(cmd string) (fullCmd string, args string) {
	parts := strings.Fields(strings.TrimSpace(cmd))
	if len(parts) == 0 {
		return "", ""
	}
	fullCmd = strings.ToUpper(parts[0])
	if len(parts) > 1 {
		args = strings.Join(parts[1:], " ")
	}
	return
}

// MatchEtcdRule 检查 etcd 命令是否匹配规则
// 规则格式: "GET", "PUT *", "DELETE config/*"
func MatchEtcdRule(rule, cmd string) bool {
	if isWildcardAll(rule) {
		cmdCmd, _ := ExtractEtcdCommand(cmd)
		return cmdCmd != ""
	}

	ruleCmd, ruleArgs := ExtractEtcdCommand(rule)
	cmdCmd, cmdArgs := ExtractEtcdCommand(cmd)

	if ruleCmd != cmdCmd {
		return false
	}
	// 无参数规则或 * 通配 → 匹配
	if ruleArgs == "" || ruleArgs == "*" {
		return true
	}
	if cmdArgs == "" {
		return false
	}
	// 按首个参数做前缀匹配（key pattern）
	ruleFirstArg := strings.Fields(ruleArgs)[0]
	cmdFirstArg := strings.Fields(cmdArgs)[0]
	matched, err := path.Match(ruleFirstArg, cmdFirstArg)
	if err != nil {
		logger.Default().Warn("etcd policy path match", zap.String("pattern", ruleFirstArg), zap.Error(err))
	}
	return matched
}

// CheckEtcdPolicy 检查 etcd 命令是否符合策略（合并默认策略后检查）
func CheckEtcdPolicy(ctx context.Context, policy *asset_entity.EtcdPolicy, cmd string) aictx.CheckResult {
	merged := EffectiveEtcdPolicy(ctx, policy)
	return checkEtcdPolicyRules(ctx, merged, cmd)
}

// checkEtcdPolicyRules 检查 etcd 命令是否符合给定策略（不合并默认策略）
func checkEtcdPolicyRules(ctx context.Context, policy *asset_entity.EtcdPolicy, cmd string) aictx.CheckResult {
	if policy == nil {
		return aictx.CheckResult{Decision: aictx.Allow, DecisionSource: aictx.SourcePolicyAllow}
	}
	// deny list 检查
	for _, rule := range policy.DenyList {
		if MatchEtcdRule(rule, cmd) {
			return aictx.CheckResult{
				Decision:       aictx.Deny,
				Message:        PolicyFmt(ctx, "etcd command denied by policy: %s", "etcd 命令被策略禁止: %s", cmd),
				DecisionSource: aictx.SourcePolicyDeny,
				MatchedPattern: rule,
			}
		}
	}
	// allow list 白名单
	if len(policy.AllowList) > 0 {
		for _, rule := range policy.AllowList {
			if MatchEtcdRule(rule, cmd) {
				return aictx.CheckResult{Decision: aictx.Allow, DecisionSource: aictx.SourcePolicyAllow}
			}
		}
		return aictx.CheckResult{Decision: aictx.NeedConfirm}
	}
	return aictx.CheckResult{Decision: aictx.Allow, DecisionSource: aictx.SourcePolicyAllow}
}
