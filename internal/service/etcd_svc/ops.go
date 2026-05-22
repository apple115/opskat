package etcd_svc

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	clientv3 "go.etcd.io/etcd/client/v3"
)

func rangeKeys(ctx context.Context, exec etcdExecutor, req EtcdRangeRequest) (EtcdRangeResponse, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = defaultEtcdPageSize
	}

	var opts []clientv3.OpOption
	opts = append(opts, clientv3.WithLimit(limit+1))
	opts = append(opts, clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend))

	key := req.Prefix
	if key == "" {
		key = req.RangeFrom
	}
	if key == "" {
		key = "\x00"
	}

	var rangeEnd string
	if req.Prefix != "" {
		rangeEnd = clientv3.GetPrefixRangeEnd(req.Prefix)
	} else if req.RangeTo != "" {
		rangeEnd = req.RangeTo
	}
	if rangeEnd != "" {
		opts = append(opts, clientv3.WithRange(rangeEnd))
	} else {
		// 查询所有 key（从 \x00 开始到末尾）
		opts = append(opts, clientv3.WithFromKey())
	}

	resp, err := exec.Get(ctx, key, opts...)
	if err != nil {
		return EtcdRangeResponse{}, fmt.Errorf("etcd range failed: %w", err)
	}

	out := EtcdRangeResponse{Count: resp.Count}
	for i, kv := range resp.Kvs {
		if i >= int(limit) {
			out.HasMore = true
			break
		}
		out.Keys = append(out.Keys, EtcdKeyValue{
			Key:            string(kv.Key),
			Value:          string(kv.Value),
			CreateRevision: kv.CreateRevision,
			ModRevision:    kv.ModRevision,
			Version:        kv.Version,
			Lease:          kv.Lease,
		})
	}
	return out, nil
}

func getKey(ctx context.Context, exec etcdExecutor, req EtcdKeyRequest) (EtcdKeyDetail, error) {
	resp, err := exec.Get(ctx, req.Key)
	if err != nil {
		return EtcdKeyDetail{}, fmt.Errorf("etcd get failed: %w", err)
	}
	if len(resp.Kvs) == 0 {
		return EtcdKeyDetail{Key: req.Key}, nil
	}
	kv := resp.Kvs[0]
	return EtcdKeyDetail{
		Key:            string(kv.Key),
		Value:          string(kv.Value),
		CreateRevision: kv.CreateRevision,
		ModRevision:    kv.ModRevision,
		Version:        kv.Version,
		Lease:          kv.Lease,
	}, nil
}

func putKey(ctx context.Context, exec etcdExecutor, req EtcdPutRequest) error {
	var opts []clientv3.OpOption
	if req.LeaseID > 0 {
		opts = append(opts, clientv3.WithLease(clientv3.LeaseID(req.LeaseID)))
	}
	_, err := exec.Put(ctx, req.Key, req.Value, opts...)
	if err != nil {
		return fmt.Errorf("etcd put failed: %w", err)
	}
	return nil
}

func deleteKeys(ctx context.Context, exec etcdExecutor, req EtcdDeleteRequest) error {
	if req.Prefix != "" {
		_, err := exec.Delete(ctx, req.Prefix, clientv3.WithPrefix())
		if err != nil {
			return fmt.Errorf("etcd delete prefix failed: %w", err)
		}
		return nil
	}
	for _, key := range req.Keys {
		_, err := exec.Delete(ctx, key)
		if err != nil {
			return fmt.Errorf("etcd delete failed: %w", err)
		}
	}
	return nil
}

func getStatus(ctx context.Context, client *clientv3.Client, assetID int64) (EtcdStatus, error) {
	endpoints := client.Endpoints()
	if len(endpoints) == 0 {
		return EtcdStatus{}, fmt.Errorf("no etcd endpoints available")
	}
	resp, err := client.Status(ctx, endpoints[0])
	if err != nil {
		return EtcdStatus{}, fmt.Errorf("etcd status failed: %w", err)
	}
	return EtcdStatus{
		Version:   resp.Version,
		DBSize:    resp.DbSize,
		Leader:    resp.Leader,
		RaftIndex: resp.RaftIndex,
		RaftTerm:  resp.RaftTerm,
	}, nil
}

func getMembers(ctx context.Context, client *clientv3.Client) ([]EtcdMember, error) {
	resp, err := client.MemberList(ctx)
	if err != nil {
		return nil, fmt.Errorf("etcd member list failed: %w", err)
	}
	out := make([]EtcdMember, 0, len(resp.Members))
	for _, m := range resp.Members {
		out = append(out, EtcdMember{
			ID:         m.ID,
			Name:       m.Name,
			PeerURLs:   m.PeerURLs,
			ClientURLs: m.ClientURLs,
			IsLearner:  m.IsLearner,
		})
	}
	return out, nil
}

// ExecuteEtcd executes a raw etcd command via the clientv3 API.
// Supported commands: get, put, delete, range
func ExecuteEtcd(ctx context.Context, client *clientv3.Client, command string) (string, error) {
	parts := splitCommand(command)
	if len(parts) == 0 {
		return "", fmt.Errorf("etcd command is empty")
	}

	cmd := strings.ToUpper(parts[0])
	switch cmd {
	case "GET":
		if len(parts) < 2 {
			return "", fmt.Errorf("GET requires a key")
		}
		resp, err := client.Get(ctx, parts[1])
		if err != nil {
			return "", err
		}
		return formatEtcdGetResponse(resp), nil

	case "PUT":
		if len(parts) < 3 {
			return "", fmt.Errorf("PUT requires key and value")
		}
		_, err := client.Put(ctx, parts[1], parts[2])
		if err != nil {
			return "", err
		}
		return `{"type":"ok","value":"OK"}`, nil

	case "DELETE":
		if len(parts) < 2 {
			return "", fmt.Errorf("DELETE requires a key")
		}
		var opts []clientv3.OpOption
		key := parts[1]
		if len(parts) > 2 && strings.ToUpper(parts[2]) == "--PREFIX" {
			opts = append(opts, clientv3.WithPrefix())
		}
		resp, err := client.Delete(ctx, key, opts...)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf(`{"type":"integer","value":%d}`, resp.Deleted), nil

	case "RANGE":
		if len(parts) < 2 {
			return "", fmt.Errorf("RANGE requires a prefix or range")
		}
		var opts []clientv3.OpOption
		key := parts[1]
		limit := int64(defaultEtcdPageSize)
		if len(parts) > 2 {
			if l, err := strconv.ParseInt(parts[2], 10, 64); err == nil && l > 0 {
				limit = l
			}
		}
		opts = append(opts, clientv3.WithLimit(limit))
		opts = append(opts, clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend))
		if len(parts) > 3 {
			opts = append(opts, clientv3.WithRange(parts[3]))
		} else {
			opts = append(opts, clientv3.WithPrefix())
		}
		resp, err := client.Get(ctx, key, opts...)
		if err != nil {
			return "", err
		}
		return formatEtcdGetResponse(resp), nil

	default:
		return "", fmt.Errorf("unsupported etcd command: %s", cmd)
	}
}

func splitCommand(command string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false
	for _, r := range command {
		switch r {
		case '"':
			inQuote = !inQuote
		case ' ', '\t':
			if inQuote {
				current.WriteRune(r)
			} else if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

func formatEtcdGetResponse(resp *clientv3.GetResponse) string {
	items := make([]map[string]any, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		items = append(items, map[string]any{
			"key":            string(kv.Key),
			"value":          string(kv.Value),
			"createRevision": kv.CreateRevision,
			"modRevision":    kv.ModRevision,
			"version":        kv.Version,
			"lease":          kv.Lease,
		})
	}
	data, _ := json.Marshal(map[string]any{
		"type":  "list",
		"count": resp.Count,
		"value": items,
	})
	return string(data)
}
