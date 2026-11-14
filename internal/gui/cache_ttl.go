package gui

import (
	"fmt"
	"proxy-checker/internal/common"
	"proxy-checker/internal/common/i18n"
	"time"

	"fyne.io/fyne/v2"
)

// refreshCacheExpiry queries the results cache for the expiry moment of each
// configured source and refreshes the TTL indicator. Safe to call from any
// goroutine: the cache read happens in the caller's goroutine, the label update
// is dispatched to the UI thread.
func (g *AppGUI) refreshCacheExpiry() {
	expiry := make(map[common.Source]time.Time, len(g.cfg.Sources))
	for _, source := range g.cfg.Sources {
		expireAt, err := g.cache.ExpireAt(source, g.cfg.Type)
		if err != nil {
			// A read failure means no trustworthy expiry for this source.
			continue
		}
		if !expireAt.IsZero() {
			expiry[source] = expireAt
		}
	}

	g.cacheExpiryMutex.Lock()
	g.cacheExpiry = expiry
	g.cacheExpiryMutex.Unlock()

	fyne.Do(func() {
		g.updateCacheTTLLabel()
	})
}

// startCacheTTLTimer launches the once-a-minute refresh of the TTL indicator.
// The timer is started once and stopped when the window closes.
func (g *AppGUI) startCacheTTLTimer() {
	if g.ttlStopChan != nil {
		return
	}
	g.ttlStopChan = make(chan struct{})

	go func(stop <-chan struct{}) {
		ticker := time.NewTicker(cacheTTLRefreshInterval)
		defer ticker.Stop()

		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				fyne.Do(func() {
					g.updateCacheTTLLabel()
				})
			}
		}
	}(g.ttlStopChan)
}

// stopCacheTTLTimer stops the once-a-minute TTL refresh goroutine.
func (g *AppGUI) stopCacheTTLTimer() {
	if g.ttlStopChan == nil {
		return
	}
	close(g.ttlStopChan)
	g.ttlStopChan = nil
}

// updateCacheTTLLabel renders the TTL indicator from the current expiry state.
// Must be called on the UI thread (directly or via fyne.Do).
func (g *AppGUI) updateCacheTTLLabel() {
	if g.cacheTTLLabel == nil {
		return
	}

	g.cacheExpiryMutex.Lock()
	nearestExpiry := nearestCacheExpiry(g.cacheExpiry)
	g.cacheExpiryMutex.Unlock()

	if nearestExpiry.IsZero() {
		g.cacheTTLLabel.Hide()
		return
	}

	g.cacheTTLLabel.Show()
	remaining := time.Until(nearestExpiry)
	if remaining <= 0 {
		g.cacheTTLLabel.SetText(i18n.T("gui.cache_ttl_expired"))
		return
	}
	g.cacheTTLLabel.SetText(fmt.Sprintf("%s %s", i18n.T("gui.cache_ttl_prefix"), formatCacheTTLRemaining(remaining)))
}

// nearestCacheExpiry returns the earliest expiry among the cached sources,
// or a zero time when the map holds no entries.
func nearestCacheExpiry(expiry map[common.Source]time.Time) time.Time {
	var nearest time.Time
	for _, expireAt := range expiry {
		if nearest.IsZero() || expireAt.Before(nearest) {
			nearest = expireAt
		}
	}
	return nearest
}

// formatCacheTTLRemaining renders the remaining cache lifetime as a compact,
// language-neutral string ("1h 05m", "12m", "<1m").
func formatCacheTTLRemaining(remaining time.Duration) string {
	if remaining <= 0 {
		return "0m"
	}

	minutes := int(remaining.Minutes())
	if minutes < 1 {
		return "<1m"
	}
	if minutes >= 60 {
		return fmt.Sprintf("%dh %02dm", minutes/60, minutes%60)
	}
	return fmt.Sprintf("%dm", minutes)
}
