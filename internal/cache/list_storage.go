package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"proxy-checker/internal/common"
	"proxy-checker/internal/fetcher"
	"proxy-checker/internal/services"
	"strings"
	"sync"
	"time"
)

// ListCacheTTL is how long a raw source list is reused before a re-fetch.
const ListCacheTTL = 4 * time.Hour

// listTypesToLoad is the order in which per-type lists are merged for "all" requests.
var listTypesToLoad = []common.ProxyType{common.ProxyHTTP, common.ProxyHTTPS, common.ProxySOCKS4, common.ProxySOCKS5}

// ListStorage persists raw proxy lists fetched from sources, separately from the
// check-results cache. Each entry is a plain text file named source-type.txt with
// one proxy per line (host, port, type, country — tab-separated); the file
// modification time serves as the freshness marker.
type ListStorage struct {
	DirPath string
	Logger  common.LoggerInterface
	mu      sync.Mutex
}

// NewListStorage creates a list cache in the temporary directory next to the results cache.
func NewListStorage(logger common.LoggerInterface) *ListStorage {
	return &ListStorage{
		DirPath: filepath.Join(os.TempDir(), common.AppName+"-lists"),
		Logger:  logger,
	}
}

// Load returns the cached raw items; services.ErrListCacheMiss signals a stale
// or absent entry. A fresh file is authoritative even when it holds an empty list.
func (s *ListStorage) Load(source common.Source, proxyType common.ProxyType) ([]*fetcher.ProxyItem, error) {
	if proxyType == common.ProxyAll {
		return s.loadAllTypes(source)
	}
	return s.loadSingle(source, proxyType)
}

// Save stores the raw items, splitting mixed-type batches into per-type files.
func (s *ListStorage) Save(source common.Source, proxyType common.ProxyType, items []*fetcher.ProxyItem) error {
	if proxyType == common.ProxyAll {
		return s.saveSplitByType(source, items)
	}
	return s.saveSingle(source, proxyType, items)
}

func (s *ListStorage) loadAllTypes(source common.Source) ([]*fetcher.ProxyItem, error) {
	var allItems []*fetcher.ProxyItem
	for _, proxyType := range listTypesToLoad {
		items, err := s.loadSingle(source, proxyType)
		if err != nil {
			// A single missing or expired type invalidates the whole batch: re-fetch.
			return nil, err
		}
		allItems = append(allItems, items...)
	}
	return deduplicateListItems(allItems), nil
}

func (s *ListStorage) loadSingle(source common.Source, proxyType common.ProxyType) ([]*fetcher.ProxyItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := listCacheFilePath(s.DirPath, source, proxyType)
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("%w: no cached list for %s:%s", services.ErrListCacheMiss, source, proxyType)
	}

	if time.Since(info.ModTime()) > ListCacheTTL {
		return nil, fmt.Errorf("%w: cached list for %s:%s is expired", services.ErrListCacheMiss, source, proxyType)
	}

	fileData, err := os.ReadFile(path)
	if err != nil {
		s.Logger.Warnf("failed to read list cache %s: %v", path, err)
		return nil, fmt.Errorf("failed to read list cache: %w", err)
	}

	return parseListItems(fileData), nil
}

func (s *ListStorage) saveSplitByType(source common.Source, items []*fetcher.ProxyItem) error {
	groups := make(map[common.ProxyType][]*fetcher.ProxyItem)
	for _, item := range items {
		groups[item.Type] = append(groups[item.Type], item)
	}

	for _, proxyType := range listTypesToLoad {
		// Every known type gets a file, even an empty one: an "all" load
		// treats a missing file as an invalid batch.
		if err := s.saveSingle(source, proxyType, groups[proxyType]); err != nil {
			return err
		}
	}
	return nil
}

func (s *ListStorage) saveSingle(source common.Source, proxyType common.ProxyType, items []*fetcher.ProxyItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(s.DirPath, 0o755); err != nil {
		wrappedErr := fmt.Errorf("failed to create list cache directory: %w", err)
		s.Logger.Warnf("%v", wrappedErr)
		return wrappedErr
	}

	path := listCacheFilePath(s.DirPath, source, proxyType)
	if err := os.WriteFile(path, serializeListItems(items), 0o600); err != nil {
		wrappedErr := fmt.Errorf("failed to write list cache %s: %w", path, err)
		s.Logger.Warnf("%v", wrappedErr)
		return wrappedErr
	}

	return nil
}

func listCacheFilePath(dirPath string, source common.Source, proxyType common.ProxyType) string {
	return filepath.Join(dirPath, string(source)+"-"+string(proxyType)+".txt")
}

func serializeListItems(items []*fetcher.ProxyItem) []byte {
	var builder strings.Builder
	for _, item := range items {
		fmt.Fprintf(&builder, "%s\t%s\t%s\t%s\n", item.Host, item.Port, item.Type, item.Country)
	}
	return []byte(builder.String())
}

func parseListItems(fileData []byte) []*fetcher.ProxyItem {
	items := make([]*fetcher.ProxyItem, 0)
	for _, line := range strings.Split(string(fileData), "\n") {
		// Trim only the newline residue: a trailing tab carries the empty country field.
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) != 4 || !common.IsKnownProxyType(common.ProxyType(parts[2])) {
			continue
		}

		items = append(items, &fetcher.ProxyItem{
			Host:    parts[0],
			Port:    parts[1],
			Type:    common.ProxyType(parts[2]),
			Country: parts[3],
		})
	}
	return items
}

func deduplicateListItems(items []*fetcher.ProxyItem) []*fetcher.ProxyItem {
	seen := make(map[string]struct{})
	result := make([]*fetcher.ProxyItem, 0, len(items))
	for _, item := range items {
		key := item.Host + ":" + item.Port
		if _, exists := seen[key]; !exists {
			seen[key] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

// ensure the adapter satisfies the pipeline port at compile time.
var _ services.ListCache = (*ListStorage)(nil)
