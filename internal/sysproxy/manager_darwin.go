package sysproxy

func NewSystemProxyManager() SystemProxyManager {
    return newMacOSProxyManager()
}
