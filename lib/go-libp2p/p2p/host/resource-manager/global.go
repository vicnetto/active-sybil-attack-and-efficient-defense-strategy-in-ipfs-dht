package rcmgr

var AllowedIps []string

func SetAllowedIpForSubnetLimit(ip string) {
	AllowedIps = []string{"127.0.0.1", ip}
}
