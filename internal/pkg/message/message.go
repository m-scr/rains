package message

import (
	"fmt"
	"sort"

	log "github.com/inconshreveable/log15"

	"github.com/britram/borat"
	"github.com/netsec-ethz/rains/internal/pkg/algorithmTypes"
	"github.com/netsec-ethz/rains/internal/pkg/keys"
	"github.com/netsec-ethz/rains/internal/pkg/sections"
	"github.com/netsec-ethz/rains/internal/pkg/signature"
	"github.com/netsec-ethz/rains/internal/pkg/token"
)

//RainsMessage represents a Message
type RainsMessage struct {
	//Capabilities is a slice of capabilities or the hash thereof which the server originating the
	//message has.
	Capabilities []Capability
	//Token is used to identify a message
	Token token.Token
	//Content is a slice of
	Content []sections.MessageSection
	//Signatures authenticate the content of this message. An encoding of RainsMessage is signed by the infrastructure key of the originating server.
	Signatures []signature.Signature
}

// MarshalCBOR writes the RAINS message to the provided writer.
// Implements the CBORMarshaler interface.
func (rm *RainsMessage) MarshalCBOR(w *borat.CBORWriter) error {
	if err := w.WriteTag(borat.CBORTag(0xE99BA8)); err != nil {
		return err
	}
	m := make(map[int]interface{})
	// A Message map MAY contain a signatures (0) key, whose value is an array
	// of Signatures over the entire message as defined in Section 5.13, to be
	// verified against the infrastructure key for the RAINS Server originating
	// the message.
	if len(rm.Signatures) > 0 {
		m[0] = rm.Signatures
	}
	// A Message map MAY contain a capabilities (1) key.
	if len(rm.Capabilities) > 0 {
		m[1] = rm.Capabilities
	}
	// A Message map MUST contain a token (2) key, whose value is a 16-byte array.
	m[2] = rm.Token
	// Message sections.
	// Each message section is a two element array [type, msgsection].
	msgsect := make([][2]interface{}, 0)
	for _, sect := range rm.Content {
		switch sect.(type) {
		case *sections.AssertionSection:
			msgsect = append(msgsect, [2]interface{}{1, sect})
		case *sections.ShardSection:
			msgsect = append(msgsect, [2]interface{}{2, sect})
		case *sections.ZoneSection:
			msgsect = append(msgsect, [2]interface{}{3, sect})
		case *sections.QuerySection:
			msgsect = append(msgsect, [2]interface{}{4, sect})
		case *sections.NotificationSection:
			msgsect = append(msgsect, [2]interface{}{23, sect})
		default:
			return fmt.Errorf("unknown section type: %T", sect)
		}
	}
	m[23] = msgsect
	return w.WriteIntMap(m)
}

func (rm *RainsMessage) UnmarshalCBOR(r *borat.CBORReader) error {
	// First read a tag to ensure we are parsing a RainsMessage
	tag, err := r.ReadTag()
	if err != nil {
		return fmt.Errorf("failed to read tag: %v", err)
	}
	if tag != borat.CBORTag(0xE99BA8) {
		return fmt.Errorf("expected tag for RAINS message but got: %v", tag)
	}
	m, err := r.ReadIntMapUntagged()
	if err != nil {
		return fmt.Errorf("failed to read map: %v", err)
	}
	// Read the signatures
	if sigs, ok := m[0]; ok {
		rm.Signatures = make([]signature.Signature, 0)
		// RAINS signatures have five common elements: the algorithm
		// identifier, a keyspace identifier, a keyphase identifier, a
		// valid-since timestamp, and a valid-until timestamp. Signatures are
		// represented as an array of these five values followed by additional
		// elements containing the signature data itself, according to the
		// algorithm identifier.
		for _, sig := range sigs.([][]interface{}) {
			alg := sig[0].(algorithmTypes.SignatureAlgorithmType)
			ks := sig[1].(keys.KeySpaceID)
			kp := sig[2].(int)
			vs := sig[3].(int64)
			vu := sig[4].(int64)
			data := sig[5]
			s := signature.Signature{
				PublicKeyID: keys.PublicKeyID{
					Algorithm: alg,
					KeySpace:  ks,
					KeyPhase:  kp,
				},
				ValidSince: vs,
				ValidUntil: vu,
				Data:       data,
			}
			rm.Signatures = append(rm.Signatures, s)
		}
	}
	// Read the capabilities
	if caps, ok := m[1]; ok {
		rm.Capabilities = make([]Capability, 0)
		for _, cap := range caps.([]interface{}) {
			rm.Capabilities = append(rm.Capabilities, Capability(cap.(string)))
		}
	}
	// read the token
	if _, ok := m[2]; !ok {
		return fmt.Errorf("token missing from RAINS message: %v", m)
	}
	for i, val := range m[2].([]interface{}) {
		rm.Token[i] = byte(val.(uint64))
	}
	// read the message sections
	for _, elem := range m[23].([]interface{}) {
		elem := elem.([]interface{})
		t := elem[0].(uint64)
		switch t {
		case 1:
			// AssertionSection
			as := &sections.AssertionSection{}
			if err := as.UnmarshalMap(elem[1].(map[int]interface{})); err != nil {
				return err
			}
			rm.Content = append(rm.Content, as)
		case 2:
			// ShardSection
			ss := &sections.ShardSection{}
			if err := ss.UnmarshalMap(elem[1].(map[int]interface{})); err != nil {
				return err
			}
			rm.Content = append(rm.Content, ss)
		case 3:
			// ZoneSection
			zs := &sections.ZoneSection{}
			if err := zs.UnmarshalMap(elem[1].(map[int]interface{})); err != nil {
				return err
			}
			rm.Content = append(rm.Content, zs)
		case 4:
			// QuerySection
			qs := &sections.QuerySection{}
			if err := qs.UnmarshalMap(elem[1].(map[int]interface{})); err != nil {
				return err
			}
			rm.Content = append(rm.Content, qs)
		case 23:
			// NotificationSection
			ns := &sections.NotificationSection{}
			if err := ns.UnmarshalMap(elem[1].(map[int]interface{})); err != nil {
				return err
			}
			rm.Content = append(rm.Content, ns)
		}
	}
	return nil
}

//Sort sorts the sections in m.Content first by Message Section Type Codes (see RAINS Protocol Specification) and
//second the sections of equal type according to their sort function.
func (m *RainsMessage) Sort() {
	var assertions []*sections.AssertionSection
	var shards []*sections.ShardSection
	var zones []*sections.ZoneSection
	var queries []*sections.QuerySection
	var addressAssertions []*sections.AddressAssertionSection
	var addressZones []*sections.AddressZoneSection
	var addressQueries []*sections.AddressQuerySection
	var notifications []*sections.NotificationSection
	for _, sec := range m.Content {
		sec.Sort()
		switch sec := sec.(type) {
		case *sections.AssertionSection:
			assertions = append(assertions, sec)
		case *sections.ShardSection:
			shards = append(shards, sec)
		case *sections.ZoneSection:
			zones = append(zones, sec)
		case *sections.QuerySection:
			queries = append(queries, sec)
		case *sections.NotificationSection:
			notifications = append(notifications, sec)
		case *sections.AddressAssertionSection:
			addressAssertions = append(addressAssertions, sec)
		case *sections.AddressZoneSection:
			addressZones = append(addressZones, sec)
		case *sections.AddressQuerySection:
			addressQueries = append(addressQueries, sec)
		default:
			log.Warn("Unsupported section type", "type", fmt.Sprintf("%T", sec))
		}
	}
	sort.Slice(assertions, func(i, j int) bool { return assertions[i].CompareTo(assertions[j]) < 0 })
	sort.Slice(shards, func(i, j int) bool { return shards[i].CompareTo(shards[j]) < 0 })
	sort.Slice(zones, func(i, j int) bool { return zones[i].CompareTo(zones[j]) < 0 })
	sort.Slice(queries, func(i, j int) bool { return queries[i].CompareTo(queries[j]) < 0 })
	sort.Slice(addressAssertions, func(i, j int) bool { return addressAssertions[i].CompareTo(addressAssertions[j]) < 0 })
	sort.Slice(addressZones, func(i, j int) bool { return addressZones[i].CompareTo(addressZones[j]) < 0 })
	sort.Slice(addressQueries, func(i, j int) bool { return addressQueries[i].CompareTo(addressQueries[j]) < 0 })
	sort.Slice(notifications, func(i, j int) bool { return notifications[i].CompareTo(notifications[j]) < 0 })
	m.Content = []sections.MessageSection{}
	for _, section := range addressQueries {
		m.Content = append(m.Content, section)
	}
	for _, section := range addressZones {
		m.Content = append(m.Content, section)
	}
	for _, section := range addressAssertions {
		m.Content = append(m.Content, section)
	}
	for _, section := range assertions {
		m.Content = append(m.Content, section)
	}
	for _, section := range shards {
		m.Content = append(m.Content, section)
	}
	for _, section := range zones {
		m.Content = append(m.Content, section)
	}
	for _, section := range queries {
		m.Content = append(m.Content, section)
	}
	for _, section := range notifications {
		m.Content = append(m.Content, section)
	}
}

//Capability is a urn of a capability
type Capability string

const (
	//NoCapability is used when the server does not listen for any connections
	NoCapability Capability = "urn:x-rains:nocapability"
	//TLSOverTCP is used when the server listens for tls over tcp connections
	TLSOverTCP Capability = "urn:x-rains:tlssrv"
)
