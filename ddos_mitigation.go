package cubepath

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// DDoSMitigationService handles communication with the DDoS Mitigation related methods of the
// CubePath API: per IP protection profiles, firewall rules on the scrubbing platform, prefix
// lists, and traffic capture.
//
// A network is a single IP ("194.26.100.205") or a CIDR ("194.26.100.0/24"). Protection
// profiles, country, ASN and prefix list filters and traffic capture need IPs with Premium
// protection; firewall rules work on any IP of the organization.
type DDoSMitigationService interface {
	// ListIPs returns the IPs of the organization with their protection status.
	ListIPs(ctx context.Context, opts *DDoSIPListOptions) (*DDoSIPList, error)
	// ListCountries returns the countries that can be used in country filters.
	ListCountries(ctx context.Context) ([]DDoSCountry, error)
	// ListASNs returns the ASNs that can be used in ASN filters. Search is an ASN number or part
	// of a name; leave it empty for every ASN.
	ListASNs(ctx context.Context, search string) ([]DDoSASN, error)

	// Protection profiles
	GetProfile(ctx context.Context, network string) (*DDoSProtectionProfile, error)
	// UpsertProfile creates or replaces the profile of an IP, or of every IP of an IPv4
	// subnet. Fields left at zero in the request are sent as zero, so start from GetProfile
	// or DefaultDDoSProtectionProfile to change only some of them.
	UpsertProfile(ctx context.Context, network string, req *DDoSProtectionProfileRequest) error
	// DeleteProfile resets the profile of an IP (or subnet) to the defaults.
	DeleteProfile(ctx context.Context, network string) error
	GetProfileCountries(ctx context.Context, ip string) ([]DDoSCountry, error)
	// SetProfileCountries replaces the countries of the profile (ISO codes such as "US").
	// Whether they are blocked or allowed depends on the profile's CountryMode.
	SetProfileCountries(ctx context.Context, ip string, isoCodes []string) error
	GetProfileASNs(ctx context.Context, ip string) ([]DDoSASN, error)
	// SetProfileASNs replaces the ASNs of the profile. Whether they are blocked or allowed
	// depends on the profile's ASNMode.
	SetProfileASNs(ctx context.Context, ip string, asns []int64) error
	GetProfilePrefixLists(ctx context.Context, ip string) ([]DDoSPrefixList, error)
	// SetProfilePrefixLists replaces the prefix lists of the profile. Whether they are
	// blocked or allowed depends on the profile's PrefixListMode.
	SetProfilePrefixLists(ctx context.Context, ip string, prefixListUUIDs []string) error

	// Firewall rules
	ListFirewallRules(ctx context.Context, network string) ([]DDoSFirewallRule, error)
	// CreateFirewallRule adds a rule. For a subnet, one rule is created for each IP.
	CreateFirewallRule(ctx context.Context, req *CreateDDoSFirewallRuleRequest) error
	DeleteFirewallRule(ctx context.Context, ruleID int) error
	// DeleteFirewallRules deletes every rule of a network for a protocol and destination port.
	DeleteFirewallRules(ctx context.Context, network string, protocol, dstPort int) error

	// Prefix lists
	ListPrefixLists(ctx context.Context) ([]DDoSPrefixList, error)
	// CreatePrefixList creates a prefix list and returns it. Names are unique per organization.
	CreatePrefixList(ctx context.Context, name, description string) (*DDoSPrefixList, error)
	DeletePrefixList(ctx context.Context, uuid string) error
	ListPrefixListEntries(ctx context.Context, uuid string) ([]string, error)
	AddPrefixListEntry(ctx context.Context, uuid, network string) error
	DeletePrefixListEntry(ctx context.Context, uuid, network string) error

	// Traffic capture
	// ListProtectedIPs returns the IPs with Premium protection, which traffic capture can query.
	ListProtectedIPs(ctx context.Context) ([]DDoSProtectedIP, error)
	// QueryTrafficCapture returns sampled packets seen by the scrubbing platform.
	QueryTrafficCapture(ctx context.Context, req *DDoSTrafficCaptureRequest) (*DDoSTrafficCapture, error)
	// GetTrafficStats returns passed and dropped traffic per time bucket.
	GetTrafficStats(ctx context.Context, req *DDoSTrafficStatsRequest) (*DDoSTrafficStats, error)
}

// DDoS firewall rule actions. Some of them only apply to one protocol (see the API docs).
const (
	DDoSActionDrop          = 0
	DDoSActionAccept        = 1
	DDoSActionFilter        = 2
	DDoSActionFiveMTCPL1    = 10
	DDoSActionFiveMTCPL2    = 11
	DDoSActionFiveMTCPL10   = 12
	DDoSActionFiveMUDPL1    = 15
	DDoSActionFiveMUDPL2    = 16
	DDoSActionFiveMUDPL3    = 17
	DDoSActionRDPTCP        = 20
	DDoSActionRDPUDP        = 21
	DDoSActionDNSUDP        = 30
	DDoSActionDNSTCP        = 31
	DDoSActionMinecraftJava = 40
	DDoSActionTLS           = 50
	DDoSActionRateLimitPPS  = 60
	DDoSActionRateLimitMbps = 61
)

// IP protocol numbers used by DDoS firewall rules.
const (
	DDoSProtocolAny  = 0
	DDoSProtocolICMP = 1
	DDoSProtocolTCP  = 6
	DDoSProtocolUDP  = 17
)

// DDoSIPListOptions filters ListIPs. IPType is "IPv4" or "IPv6".
type DDoSIPListOptions struct {
	IPType     string
	Location   string
	HasProfile *bool
}

// DDoSSingleIP is a single IP with its protection status. ProtectionType is "Basic",
// "Premium" or "Premium Always-On".
type DDoSSingleIP struct {
	Network             string `json:"network"`
	IPType              string `json:"ip_type"`
	ProtectionType      string `json:"protection_type"`
	LocationName        string `json:"location_name"`
	LocationDescription string `json:"location_description"`
	HasProfile          bool   `json:"has_profile"`
	FirewallRulesCount  int    `json:"firewall_rules_count"`
}

// DDoSSubnetIP is one IP of a subnet.
type DDoSSubnetIP struct {
	Address            string `json:"address"`
	HasProfile         bool   `json:"has_profile"`
	FirewallRulesCount int    `json:"firewall_rules_count"`
}

// DDoSSubnet is a subnet with its protection status and, for IPv4, its IPs.
type DDoSSubnet struct {
	Network             string         `json:"network"`
	Prefix              int            `json:"prefix"`
	IPType              string         `json:"ip_type"`
	ProtectionType      string         `json:"protection_type"`
	LocationName        string         `json:"location_name"`
	LocationDescription string         `json:"location_description"`
	HasProfile          bool           `json:"has_profile"`
	FirewallRulesCount  int            `json:"firewall_rules_count"`
	IPAddresses         []DDoSSubnetIP `json:"ip_addresses"`
}

// DDoSIPList is the list of IPs of the organization, split into single IPs and subnets.
type DDoSIPList struct {
	SingleIPs []DDoSSingleIP `json:"single_ips"`
	Subnets   []DDoSSubnet   `json:"subnets"`
	Total     int            `json:"total"`
}

// DDoSCountry is a country usable in country filters.
type DDoSCountry struct {
	ISOCode string `json:"iso_code"`
	Name    string `json:"name"`
}

// DDoSASN is an ASN usable in ASN filters.
type DDoSASN struct {
	ASN  int64  `json:"asn"`
	Name string `json:"name"`
}

// DDoSProtectionProfileRequest holds the settings of a protection profile.
//
// Levels go from 0 (off) to 10. TCPValidationLevel and TCPValidationSymLevel are mutually
// exclusive. DefaultAction is 0 (filter), 1 (accept) or 2 (drop). CountryMode, ASNMode and
// PrefixListMode are 0 (off), 1 (block the listed ones) or 2 (allow only the listed ones).
// Rate limits must be at least 1. AlwaysOnMitigation and SymmetricRouting (0 or 1) are only
// changed when set.
type DDoSProtectionProfileRequest struct {
	TCPValidationLevel    int  `json:"tcp_validation_level"`
	TCPValidationSymLevel int  `json:"tcp_validation_sym_level"`
	UDPValidationLevel    int  `json:"udp_validation_level"`
	InvalidFilterLevel    int  `json:"invalid_filter_level"`
	FragmentedFilterLevel int  `json:"fragmented_filter_level"`
	AmplificationUDPLevel int  `json:"amplification_udp_level"`
	AmplificationTCPLevel int  `json:"amplification_tcp_level"`
	ICMPRateLimitLevel    int  `json:"icmp_rate_limit_level"`
	SamePacketSizeLevel   int  `json:"same_packet_size_level"`
	StatefulFirewallLevel int  `json:"stateful_firewall_level"`
	DefaultAction         int  `json:"default_action"`
	CountryMode           int  `json:"country_mode"`
	ASNMode               int  `json:"asn_mode"`
	PrefixListMode        int  `json:"prefix_list_mode"`
	UDPThresholdPPS       int  `json:"udp_threshold_pps"`
	TCPThresholdPPS       int  `json:"tcp_threshold_pps"`
	TCPSynThresholdPPS    int  `json:"tcp_syn_threshold_pps"`
	TCPAckThresholdPPS    int  `json:"tcp_ack_threshold_pps"`
	ICMPThresholdPPS      int  `json:"icmp_threshold_pps"`
	UDPThresholdMbps      int  `json:"udp_threshold_mbps"`
	TCPThresholdMbps      int  `json:"tcp_threshold_mbps"`
	TCPSynThresholdMbps   int  `json:"tcp_syn_threshold_mbps"`
	TCPAckThresholdMbps   int  `json:"tcp_ack_threshold_mbps"`
	ICMPThresholdMbps     int  `json:"icmp_threshold_mbps"`
	SynFloodThreshold     int  `json:"syn_flood_threshold"`
	SynFloodBlockSecs     int  `json:"syn_flood_block_secs"`
	AlwaysOnMitigation    *int `json:"always_on_mitigation,omitempty"`
	SymmetricRouting      *int `json:"symmetric_routing,omitempty"`
}

// DefaultDDoSProtectionProfile returns the settings the API uses for an IP without profile.
func DefaultDDoSProtectionProfile() *DDoSProtectionProfileRequest {
	return &DDoSProtectionProfileRequest{
		TCPValidationLevel:    1,
		InvalidFilterLevel:    1,
		FragmentedFilterLevel: 1,
		AmplificationUDPLevel: 1,
		AmplificationTCPLevel: 1,
		SamePacketSizeLevel:   1,
		UDPThresholdPPS:       1000,
		TCPThresholdPPS:       1000,
		TCPSynThresholdPPS:    10,
		TCPAckThresholdPPS:    200,
		ICMPThresholdPPS:      100,
		UDPThresholdMbps:      100,
		TCPThresholdMbps:      100,
		TCPSynThresholdMbps:   100,
		TCPAckThresholdMbps:   100,
		ICMPThresholdMbps:     100,
		SynFloodBlockSecs:     60,
	}
}

// DDoSProtectionProfile is the protection profile of an IP or network.
type DDoSProtectionProfile struct {
	Network               string `json:"network"`
	TCPValidationLevel    int    `json:"tcp_validation_level"`
	TCPValidationSymLevel int    `json:"tcp_validation_sym_level"`
	UDPValidationLevel    int    `json:"udp_validation_level"`
	InvalidFilterLevel    int    `json:"invalid_filter_level"`
	FragmentedFilterLevel int    `json:"fragmented_filter_level"`
	AmplificationUDPLevel int    `json:"amplification_udp_level"`
	AmplificationTCPLevel int    `json:"amplification_tcp_level"`
	ICMPRateLimitLevel    int    `json:"icmp_rate_limit_level"`
	SamePacketSizeLevel   int    `json:"same_packet_size_level"`
	StatefulFirewallLevel int    `json:"stateful_firewall_level"`
	DefaultAction         int    `json:"default_action"`
	CountryMode           int    `json:"country_mode"`
	ASNMode               int    `json:"asn_mode"`
	PrefixListMode        int    `json:"prefix_list_mode"`
	UDPThresholdPPS       int    `json:"udp_threshold_pps"`
	TCPThresholdPPS       int    `json:"tcp_threshold_pps"`
	TCPSynThresholdPPS    int    `json:"tcp_syn_threshold_pps"`
	TCPAckThresholdPPS    int    `json:"tcp_ack_threshold_pps"`
	ICMPThresholdPPS      int    `json:"icmp_threshold_pps"`
	UDPThresholdMbps      int    `json:"udp_threshold_mbps"`
	TCPThresholdMbps      int    `json:"tcp_threshold_mbps"`
	TCPSynThresholdMbps   int    `json:"tcp_syn_threshold_mbps"`
	TCPAckThresholdMbps   int    `json:"tcp_ack_threshold_mbps"`
	ICMPThresholdMbps     int    `json:"icmp_threshold_mbps"`
	SynFloodThreshold     int    `json:"syn_flood_threshold"`
	SynFloodBlockSecs     int    `json:"syn_flood_block_secs"`
	AlwaysOnMitigation    int    `json:"always_on_mitigation"`
	SymmetricRouting      int    `json:"symmetric_routing"`
}

// DDoSFirewallRule is a firewall rule on the scrubbing platform. The rate limit fields are
// used by the rate limit actions (60 in packets per second, 61 in Mbps).
type DDoSFirewallRule struct {
	ID          int    `json:"id"`
	Network     string `json:"network"`
	Protocol    int    `json:"protocol"`
	DstPort     int    `json:"dst_port"`
	Action      int    `json:"action"`
	ActionLabel string `json:"action_label"`
	TCPSyn      int    `json:"tcp_syn"`
	TCPAck      int    `json:"tcp_ack"`
	TCPSynAck   int    `json:"tcp_synack"`
	TCPRst      int    `json:"tcp_rst"`
	TCPFin      int    `json:"tcp_fin"`
	TCPAll      int    `json:"tcp_all"`
	UDP         int    `json:"udp"`
	ICMP        int    `json:"icmp"`
}

// CreateDDoSFirewallRuleRequest represents a request to create a DDoS firewall rule.
// Protocol is one of the DDoSProtocol constants, DstPort 0 means any port and Action is one
// of the DDoSAction constants.
type CreateDDoSFirewallRuleRequest struct {
	Network   string `json:"network"`
	Protocol  int    `json:"protocol"`
	DstPort   int    `json:"dst_port"`
	Action    int    `json:"action"`
	TCPSyn    int    `json:"tcp_syn,omitempty"`
	TCPAck    int    `json:"tcp_ack,omitempty"`
	TCPSynAck int    `json:"tcp_synack,omitempty"`
	TCPRst    int    `json:"tcp_rst,omitempty"`
	TCPFin    int    `json:"tcp_fin,omitempty"`
	TCPAll    int    `json:"tcp_all,omitempty"`
	UDP       int    `json:"udp,omitempty"`
	ICMP      int    `json:"icmp,omitempty"`
}

// DDoSPrefixList is a list of networks usable in profile filters. Global lists are provided
// by CubePath and cannot be changed.
type DDoSPrefixList struct {
	UUID         string `json:"uuid"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	IsGlobal     bool   `json:"is_global"`
	CreatedAt    string `json:"created_at,omitempty"`
	EntriesCount int    `json:"entries_count"`
}

// DDoSProtectedIP is an IP with Premium protection.
type DDoSProtectedIP struct {
	Address    string `json:"address"`
	Netmask    string `json:"netmask"`
	Network    string `json:"network"`
	Location   string `json:"location"`
	AssignedTo string `json:"assigned_to"`
}

// DDoSTrafficCaptureRequest filters a traffic capture query. DestinationIP (an IP or subnet of
// the organization) and the time window are required. Protocols are names such as "TCP",
// actions "PASS" or "DROP".
type DDoSTrafficCaptureRequest struct {
	StartTime        time.Time `json:"start_time"`
	EndTime          time.Time `json:"end_time"`
	DestinationIP    string    `json:"destination_ip"`
	IncludeSrcIPs    []string  `json:"include_src_ips,omitempty"`
	ExcludeSrcIPs    []string  `json:"exclude_src_ips,omitempty"`
	IncludeSrcPorts  []int     `json:"include_src_ports,omitempty"`
	ExcludeSrcPorts  []int     `json:"exclude_src_ports,omitempty"`
	IncludeDstPorts  []int     `json:"include_dst_ports,omitempty"`
	ExcludeDstPorts  []int     `json:"exclude_dst_ports,omitempty"`
	MinSrcPort       *int      `json:"min_src_port,omitempty"`
	MaxSrcPort       *int      `json:"max_src_port,omitempty"`
	MinDstPort       *int      `json:"min_dst_port,omitempty"`
	MaxDstPort       *int      `json:"max_dst_port,omitempty"`
	IncludeProtocols []string  `json:"include_protocols,omitempty"`
	ExcludeProtocols []string  `json:"exclude_protocols,omitempty"`
	IncludeActions   []string  `json:"include_actions,omitempty"`
	ExcludeActions   []string  `json:"exclude_actions,omitempty"`
	IncludeTCPFlags  []string  `json:"include_tcp_flags,omitempty"`
	ExcludeTCPFlags  []string  `json:"exclude_tcp_flags,omitempty"`
	MinPacketLen     *int      `json:"min_packet_len,omitempty"`
	MaxPacketLen     *int      `json:"max_packet_len,omitempty"`
	MinTTL           *int      `json:"min_ttl,omitempty"`
	MaxTTL           *int      `json:"max_ttl,omitempty"`
	HasPayload       *bool     `json:"has_payload,omitempty"`
	Limit            int       `json:"limit,omitempty"`
}

// DDoSTrafficLog is one sampled packet.
type DDoSTrafficLog struct {
	Timestamp      string  `json:"timestamp"`
	Node           *string `json:"node"`
	SrcIP          string  `json:"src_ip"`
	DstIP          string  `json:"dst_ip"`
	SrcPort        int     `json:"src_port"`
	DstPort        int     `json:"dst_port"`
	Protocol       string  `json:"protocol"`
	Action         string  `json:"action"`
	MitigationName *string `json:"mitigation_name"`
	IsDrop         bool    `json:"is_drop"`
	PacketLen      int     `json:"packet_len"`
	TTL            int     `json:"ttl"`
	SampleRate     *int    `json:"sample_rate"`
	TCPFlags       *string `json:"tcp_flags"`
	ICMPType       *int    `json:"icmp_type"`
	ICMPCode       *int    `json:"icmp_code"`
	SrcCountry     *string `json:"src_country"`
	PayloadLen     *int    `json:"payload_len"`
	Payload        *string `json:"payload"`
}

// DDoSTrafficCapture is the result of a traffic capture query.
type DDoSTrafficCapture struct {
	StartTime string           `json:"start_time"`
	EndTime   string           `json:"end_time"`
	TotalLogs int              `json:"total_logs"`
	Logs      []DDoSTrafficLog `json:"logs"`
}

// DDoSTrafficStatsRequest selects the traffic stats. Leave DestinationIPs empty for every IP of
// the organization. Interval is "10s", "30s", "1m" (default), "5m", "15m" or "1h".
type DDoSTrafficStatsRequest struct {
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
	DestinationIPs []string  `json:"destination_ips,omitempty"`
	Interval       string    `json:"interval,omitempty"`
}

// DDoSTrafficStatsBucket is the passed and dropped traffic of one time bucket.
type DDoSTrafficStatsBucket struct {
	Timestamp string  `json:"timestamp"`
	PassCount int64   `json:"pass_count"`
	DropCount int64   `json:"drop_count"`
	PassBytes int64   `json:"pass_bytes"`
	DropBytes int64   `json:"drop_bytes"`
	PassPPS   float64 `json:"pass_pps"`
	DropPPS   float64 `json:"drop_pps"`
}

// DDoSTrafficStats is the passed and dropped traffic per time bucket.
type DDoSTrafficStats struct {
	StartTime string                   `json:"start_time"`
	EndTime   string                   `json:"end_time"`
	Interval  string                   `json:"interval"`
	TotalPass int64                    `json:"total_pass"`
	TotalDrop int64                    `json:"total_drop"`
	Buckets   []DDoSTrafficStatsBucket `json:"buckets"`
}

type ddosMitigationService struct {
	client *Client
}

// networkPath escapes each part of an IP or CIDR for a path; the "/" of a CIDR is kept, the
// API accepts it literally.
func networkPath(network string) string {
	parts := strings.Split(network, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return strings.Join(parts, "/")
}

func (s *ddosMitigationService) ListIPs(ctx context.Context, opts *DDoSIPListOptions) (*DDoSIPList, error) {
	path := "/ddos-mitigation/ips"
	if opts != nil {
		v := url.Values{}
		if opts.IPType != "" {
			v.Set("ip_type", opts.IPType)
		}
		if opts.Location != "" {
			v.Set("location", opts.Location)
		}
		if opts.HasProfile != nil {
			v.Set("has_profile", strconv.FormatBool(*opts.HasProfile))
		}
		if len(v) > 0 {
			path += "?" + v.Encode()
		}
	}
	var result DDoSIPList
	if err := s.client.get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *ddosMitigationService) ListCountries(ctx context.Context) ([]DDoSCountry, error) {
	var result struct {
		Countries []DDoSCountry `json:"countries"`
	}
	if err := s.client.get(ctx, "/ddos-mitigation/countries", &result); err != nil {
		return nil, err
	}
	return result.Countries, nil
}

func (s *ddosMitigationService) ListASNs(ctx context.Context, search string) ([]DDoSASN, error) {
	path := "/ddos-mitigation/asns"
	if search != "" {
		path += "?search=" + url.QueryEscape(search)
	}
	var result struct {
		ASNs []DDoSASN `json:"asns"`
	}
	if err := s.client.get(ctx, path, &result); err != nil {
		return nil, err
	}
	return result.ASNs, nil
}

func (s *ddosMitigationService) GetProfile(ctx context.Context, network string) (*DDoSProtectionProfile, error) {
	var profile DDoSProtectionProfile
	if err := s.client.get(ctx, "/ddos-mitigation/profiles/"+networkPath(network), &profile); err != nil {
		return nil, err
	}
	return &profile, nil
}

func (s *ddosMitigationService) UpsertProfile(ctx context.Context, network string, req *DDoSProtectionProfileRequest) error {
	return s.client.put(ctx, "/ddos-mitigation/profiles/"+networkPath(network), req, nil)
}

func (s *ddosMitigationService) DeleteProfile(ctx context.Context, network string) error {
	return s.client.del(ctx, "/ddos-mitigation/profiles/"+networkPath(network))
}

func (s *ddosMitigationService) GetProfileCountries(ctx context.Context, ip string) ([]DDoSCountry, error) {
	var result struct {
		Countries []DDoSCountry `json:"countries"`
	}
	if err := s.client.get(ctx, fmt.Sprintf("/ddos-mitigation/profiles/%s/countries", url.PathEscape(ip)), &result); err != nil {
		return nil, err
	}
	return result.Countries, nil
}

func (s *ddosMitigationService) SetProfileCountries(ctx context.Context, ip string, isoCodes []string) error {
	if isoCodes == nil {
		isoCodes = []string{}
	}
	body := map[string]interface{}{
		"iso_codes": isoCodes,
	}
	return s.client.put(ctx, fmt.Sprintf("/ddos-mitigation/profiles/%s/countries", url.PathEscape(ip)), body, nil)
}

func (s *ddosMitigationService) GetProfileASNs(ctx context.Context, ip string) ([]DDoSASN, error) {
	var result struct {
		ASNs []DDoSASN `json:"asns"`
	}
	if err := s.client.get(ctx, fmt.Sprintf("/ddos-mitigation/profiles/%s/asns", url.PathEscape(ip)), &result); err != nil {
		return nil, err
	}
	return result.ASNs, nil
}

func (s *ddosMitigationService) SetProfileASNs(ctx context.Context, ip string, asns []int64) error {
	if asns == nil {
		asns = []int64{}
	}
	body := map[string]interface{}{
		"asns": asns,
	}
	return s.client.put(ctx, fmt.Sprintf("/ddos-mitigation/profiles/%s/asns", url.PathEscape(ip)), body, nil)
}

func (s *ddosMitigationService) GetProfilePrefixLists(ctx context.Context, ip string) ([]DDoSPrefixList, error) {
	var result struct {
		PrefixLists []DDoSPrefixList `json:"prefix_lists"`
	}
	if err := s.client.get(ctx, fmt.Sprintf("/ddos-mitigation/profiles/%s/prefix-lists", url.PathEscape(ip)), &result); err != nil {
		return nil, err
	}
	return result.PrefixLists, nil
}

func (s *ddosMitigationService) SetProfilePrefixLists(ctx context.Context, ip string, prefixListUUIDs []string) error {
	if prefixListUUIDs == nil {
		prefixListUUIDs = []string{}
	}
	body := map[string]interface{}{
		"uuids": prefixListUUIDs,
	}
	return s.client.put(ctx, fmt.Sprintf("/ddos-mitigation/profiles/%s/prefix-lists", url.PathEscape(ip)), body, nil)
}

func (s *ddosMitigationService) ListFirewallRules(ctx context.Context, network string) ([]DDoSFirewallRule, error) {
	var result struct {
		Rules []DDoSFirewallRule `json:"rules"`
	}
	if err := s.client.get(ctx, "/ddos-mitigation/firewall-rules/"+networkPath(network), &result); err != nil {
		return nil, err
	}
	return result.Rules, nil
}

func (s *ddosMitigationService) CreateFirewallRule(ctx context.Context, req *CreateDDoSFirewallRuleRequest) error {
	return s.client.post(ctx, "/ddos-mitigation/firewall-rules", req, nil)
}

func (s *ddosMitigationService) DeleteFirewallRule(ctx context.Context, ruleID int) error {
	return s.client.del(ctx, fmt.Sprintf("/ddos-mitigation/firewall-rules/%d", ruleID))
}

func (s *ddosMitigationService) DeleteFirewallRules(ctx context.Context, network string, protocol, dstPort int) error {
	v := url.Values{}
	v.Set("network", network)
	v.Set("protocol", strconv.Itoa(protocol))
	v.Set("dst_port", strconv.Itoa(dstPort))
	return s.client.del(ctx, "/ddos-mitigation/firewall-rules/bulk?"+v.Encode())
}

func (s *ddosMitigationService) ListPrefixLists(ctx context.Context) ([]DDoSPrefixList, error) {
	var result struct {
		PrefixLists []DDoSPrefixList `json:"prefix_lists"`
	}
	if err := s.client.get(ctx, "/ddos-mitigation/prefix-lists", &result); err != nil {
		return nil, err
	}
	return result.PrefixLists, nil
}

// CreatePrefixList creates the list, then reads it back by name: the API only answers with a
// message.
func (s *ddosMitigationService) CreatePrefixList(ctx context.Context, name, description string) (*DDoSPrefixList, error) {
	body := map[string]interface{}{
		"name": name,
	}
	if description != "" {
		body["description"] = description
	}
	if err := s.client.post(ctx, "/ddos-mitigation/prefix-lists", body, nil); err != nil {
		return nil, err
	}
	lists, err := s.ListPrefixLists(ctx)
	if err != nil {
		return nil, err
	}
	for i := range lists {
		if lists[i].Name == name && !lists[i].IsGlobal {
			return &lists[i], nil
		}
	}
	return nil, &APIError{StatusCode: 404, Message: "Not Found", Detail: fmt.Sprintf("prefix list %q not found after creation", name)}
}

func (s *ddosMitigationService) DeletePrefixList(ctx context.Context, uuid string) error {
	return s.client.del(ctx, "/ddos-mitigation/prefix-lists/"+url.PathEscape(uuid))
}

func (s *ddosMitigationService) ListPrefixListEntries(ctx context.Context, uuid string) ([]string, error) {
	var entries []struct {
		Network string `json:"network"`
	}
	if err := s.client.get(ctx, fmt.Sprintf("/ddos-mitigation/prefix-lists/%s/entries", url.PathEscape(uuid)), &entries); err != nil {
		return nil, err
	}
	networks := make([]string, 0, len(entries))
	for _, e := range entries {
		networks = append(networks, e.Network)
	}
	return networks, nil
}

func (s *ddosMitigationService) AddPrefixListEntry(ctx context.Context, uuid, network string) error {
	body := map[string]interface{}{
		"network": network,
	}
	return s.client.post(ctx, fmt.Sprintf("/ddos-mitigation/prefix-lists/%s/entries", url.PathEscape(uuid)), body, nil)
}

func (s *ddosMitigationService) DeletePrefixListEntry(ctx context.Context, uuid, network string) error {
	return s.client.del(ctx, fmt.Sprintf("/ddos-mitigation/prefix-lists/%s/entries/%s", url.PathEscape(uuid), networkPath(network)))
}

func (s *ddosMitigationService) ListProtectedIPs(ctx context.Context) ([]DDoSProtectedIP, error) {
	var result struct {
		IPs []DDoSProtectedIP `json:"ips"`
	}
	if err := s.client.get(ctx, "/ddos-mitigation/traffic-capture/protected-ips", &result); err != nil {
		return nil, err
	}
	return result.IPs, nil
}

func (s *ddosMitigationService) QueryTrafficCapture(ctx context.Context, req *DDoSTrafficCaptureRequest) (*DDoSTrafficCapture, error) {
	var result DDoSTrafficCapture
	if err := s.client.post(ctx, "/ddos-mitigation/traffic-capture", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *ddosMitigationService) GetTrafficStats(ctx context.Context, req *DDoSTrafficStatsRequest) (*DDoSTrafficStats, error) {
	var result DDoSTrafficStats
	if err := s.client.post(ctx, "/ddos-mitigation/traffic-capture/stats", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
