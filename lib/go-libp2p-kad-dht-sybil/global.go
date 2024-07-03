package dht

var EclipsedCid string
var AllowedIps []string
var IsActive bool

var AuthorizeAll = "0.0.0.0"
var localGroup = "127.0.0.0"

func SetAttackConfiguration(cid string, isActiveMode bool) {
	EclipsedCid = cid
	IsActive = isActiveMode
}

func SetGroupToBypassDiversityFilter(ip string) {
	AllowedIps = []string{localGroup, ip}
}
