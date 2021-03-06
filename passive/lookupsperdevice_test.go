package passive

import (
	"fmt"

	"code.google.com/p/goprotobuf/proto"
	"github.com/sburnett/lexicographic-tuples"
	"github.com/sburnett/transformer"
	"github.com/sburnett/transformer/store"
)

func runLookupsPerDevicePipeline(traces map[string]Trace, consistentRanges []*store.Record, addressIdToMac map[string]string) {
	levelDbManager := store.NewSliceManager()

	tracesStore := levelDbManager.Writer("traces")
	tracesStore.BeginWriting()
	for encodedKey, trace := range traces {
		encodedTrace, err := proto.Marshal(&trace)
		if err != nil {
			panic(fmt.Errorf("Error encoding protocol buffer: %v", err))
		}
		tracesStore.WriteRecord(&store.Record{Key: []byte(encodedKey), Value: encodedTrace})
	}
	tracesStore.EndWriting()

	availabilityIntervalsStore := levelDbManager.Writer("consistent-ranges")
	availabilityIntervalsStore.BeginWriting()
	for _, record := range consistentRanges {
		availabilityIntervalsStore.WriteRecord(record)
	}
	availabilityIntervalsStore.EndWriting()

	addressIdStore := levelDbManager.Writer("bytesperdomain-address-id-table")
	addressIdStore.BeginWriting()
	for encodedKey, encodedValue := range addressIdToMac {
		addressIdStore.WriteRecord(&store.Record{Key: []byte(encodedKey), Value: []byte(encodedValue)})
	}
	addressIdStore.EndWriting()

	transformer.RunPipeline(LookupsPerDevicePipeline(levelDbManager))

	fmt.Printf("LookupsPerDevice:\n")
	lookupsPerDeviceStore := levelDbManager.Reader("lookupsperdevice-lookups-per-device")
	lookupsPerDeviceStore.BeginReading()
	for {
		record, err := lookupsPerDeviceStore.ReadRecord()
		if err != nil {
			panic(err)
		}
		if record == nil {
			break
		}
		var (
			nodeId, macAddress, domain string
			count                      int64
		)
		lex.DecodeOrDie(record.Key, &nodeId, &macAddress, &domain)
		lex.DecodeOrDie(record.Value, &count)
		fmt.Printf("%s,%s,%s: %d\n", nodeId, macAddress, domain, count)
	}
	lookupsPerDeviceStore.EndReading()

	fmt.Printf("\nLookupsPerDevicePerHour:\n")
	lookupsPerDevicePerHourStore := levelDbManager.Reader("lookupsperdevice-lookups-per-device-per-hour")
	lookupsPerDevicePerHourStore.BeginReading()
	for {
		record, err := lookupsPerDevicePerHourStore.ReadRecord()
		if err != nil {
			panic(err)
		}
		if record == nil {
			break
		}
		var (
			nodeId, macAddress, domain string
			timestamp, count           int64
		)
		lex.DecodeOrDie(record.Key, &nodeId, &macAddress, &domain, &timestamp)
		lex.DecodeOrDie(record.Value, &count)
		fmt.Printf("%s,%s,%s,%d: %d\n", nodeId, macAddress, domain, timestamp, count)
	}
	lookupsPerDevicePerHourStore.EndReading()
}

func ExampleLookupsPerDevice_empty() {
	traces := map[string]Trace{}
	consistentRanges := []*store.Record{}
	addressIdStore := map[string]string{}

	runLookupsPerDevicePipeline(traces, consistentRanges, addressIdStore)

	// Output:
	// LookupsPerDevice:
	//
	// LookupsPerDevicePerHour:
}

func ExampleLookupsPerDevice_oneDomain() {
	trace := Trace{
		ARecord: []*DnsARecord{
			&DnsARecord{
				AddressId:  proto.Int32(0),
				Anonymized: proto.Bool(false),
				Domain:     proto.String("m.domain"),
			},
		},
	}
	traces := map[string]Trace{
		string(lex.EncodeOrDie("node1", "anon1", int64(0), int32(0))): trace,
	}
	consistentRanges := []*store.Record{
		&store.Record{
			Key:   lex.EncodeOrDie("node1", "anon1", int64(0), int32(0)),
			Value: lex.EncodeOrDie("node1", "anon1", int64(0), int32(0)),
		},
	}
	addressIdStore := map[string]string{
		string(lex.EncodeOrDie("node1", "anon1", int64(0), int32(0), int32(0))): string(lex.EncodeOrDie("mac1")),
	}

	runLookupsPerDevicePipeline(traces, consistentRanges, addressIdStore)

	// Output:
	// LookupsPerDevice:
	// node1,mac1,m.domain: 1
	//
	// LookupsPerDevicePerHour:
	// node1,mac1,m.domain,0: 1
}

func ExampleLookupsPerDevice_oneCname() {
	trace := Trace{
		CnameRecord: []*DnsCnameRecord{
			&DnsCnameRecord{
				AddressId:        proto.Int32(0),
				DomainAnonymized: proto.Bool(false),
				Domain:           proto.String("m.domain1"),
				CnameAnonymized:  proto.Bool(false),
				Cname:            proto.String("m.domain2"),
			},
		},
	}
	traces := map[string]Trace{
		string(lex.EncodeOrDie("node1", "anon1", int64(0), int32(0))): trace,
	}
	consistentRanges := []*store.Record{
		&store.Record{
			Key:   lex.EncodeOrDie("node1", "anon1", int64(0), int32(0)),
			Value: lex.EncodeOrDie("node1", "anon1", int64(0), int32(0)),
		},
	}
	addressIdStore := map[string]string{
		string(lex.EncodeOrDie("node1", "anon1", int64(0), int32(0), int32(0))): string(lex.EncodeOrDie("mac1")),
	}

	runLookupsPerDevicePipeline(traces, consistentRanges, addressIdStore)

	// Output:
	// LookupsPerDevice:
	// node1,mac1,m.domain1: 1
	// node1,mac1,m.domain2: 1
	//
	// LookupsPerDevicePerHour:
	// node1,mac1,m.domain1,0: 1
	// node1,mac1,m.domain2,0: 1
}

func ExampleLookupsPerDevice_matchDomain() {
	trace := Trace{
		ARecord: []*DnsARecord{
			&DnsARecord{
				AddressId:  proto.Int32(0),
				Anonymized: proto.Bool(false),
				Domain:     proto.String("m.domain"),
			},
			&DnsARecord{
				AddressId:  proto.Int32(0),
				Anonymized: proto.Bool(false),
				Domain:     proto.String("x.m.domain"),
			},
			&DnsARecord{
				AddressId:  proto.Int32(0),
				Anonymized: proto.Bool(false),
				Domain:     proto.String("mdomain"),
			},
		},
	}
	traces := map[string]Trace{
		string(lex.EncodeOrDie("node1", "anon1", int64(0), int32(0))): trace,
	}
	consistentRanges := []*store.Record{
		&store.Record{
			Key:   lex.EncodeOrDie("node1", "anon1", int64(0), int32(0)),
			Value: lex.EncodeOrDie("node1", "anon1", int64(0), int32(0)),
		},
	}
	addressIdStore := map[string]string{
		string(lex.EncodeOrDie("node1", "anon1", int64(0), int32(0), int32(0))): string(lex.EncodeOrDie("mac1")),
	}

	runLookupsPerDevicePipeline(traces, consistentRanges, addressIdStore)

	// Output:
	// LookupsPerDevice:
	// node1,mac1,m.domain: 1
	// node1,mac1,x.m.domain: 1
	//
	// LookupsPerDevicePerHour:
	// node1,mac1,m.domain,0: 1
	// node1,mac1,x.m.domain,0: 1
}

func ExampleLookupsPerDevice_multipleAddresses() {
	trace := Trace{
		ARecord: []*DnsARecord{
			&DnsARecord{
				AddressId:  proto.Int32(0),
				Anonymized: proto.Bool(false),
				Domain:     proto.String("m.domain1"),
			},
			&DnsARecord{
				AddressId:  proto.Int32(1),
				Anonymized: proto.Bool(false),
				Domain:     proto.String("m.domain2"),
			},
		},
	}
	traces := map[string]Trace{
		string(lex.EncodeOrDie("node1", "anon1", int64(0), int32(0))): trace,
	}
	consistentRanges := []*store.Record{
		&store.Record{
			Key:   lex.EncodeOrDie("node1", "anon1", int64(0), int32(0)),
			Value: lex.EncodeOrDie("node1", "anon1", int64(0), int32(0)),
		},
	}
	addressIdStore := map[string]string{
		string(lex.EncodeOrDie("node1", "anon1", int64(0), int32(0), int32(0))): string(lex.EncodeOrDie("mac1")),
		string(lex.EncodeOrDie("node1", "anon1", int64(0), int32(1), int32(0))): string(lex.EncodeOrDie("mac2")),
	}

	runLookupsPerDevicePipeline(traces, consistentRanges, addressIdStore)

	// Output:
	// LookupsPerDevice:
	// node1,mac1,m.domain1: 1
	// node1,mac2,m.domain2: 1
	//
	// LookupsPerDevicePerHour:
	// node1,mac1,m.domain1,0: 1
	// node1,mac2,m.domain2,0: 1
}

func ExampleLookupsPerDevice_anonymization() {
	trace := Trace{
		ARecord: []*DnsARecord{
			&DnsARecord{
				AddressId:  proto.Int32(0),
				Anonymized: proto.Bool(true),
				Domain:     proto.String("m.domain"),
			},
		},
		CnameRecord: []*DnsCnameRecord{
			&DnsCnameRecord{
				AddressId:        proto.Int32(0),
				DomainAnonymized: proto.Bool(true),
				Domain:           proto.String("m.domain1"),
				CnameAnonymized:  proto.Bool(true),
				Cname:            proto.String("m.domain2"),
			},
		},
	}
	traces := map[string]Trace{
		string(lex.EncodeOrDie("node1", "anon1", int64(0), int32(0))): trace,
	}
	consistentRanges := []*store.Record{
		&store.Record{
			Key:   lex.EncodeOrDie("node1", "anon1", int64(0), int32(0)),
			Value: lex.EncodeOrDie("node1", "anon1", int64(0), int32(0)),
		},
	}
	addressIdStore := map[string]string{
		string(lex.EncodeOrDie("node1", "anon1", int64(0), int32(0), int32(0))): string(lex.EncodeOrDie("mac1")),
	}

	runLookupsPerDevicePipeline(traces, consistentRanges, addressIdStore)

	// Output:
	// LookupsPerDevice:
	//
	// LookupsPerDevicePerHour:
}

func ExampleLookupsPerDevice_multipleLookups() {
	trace := Trace{
		ARecord: []*DnsARecord{
			&DnsARecord{
				AddressId:  proto.Int32(0),
				Anonymized: proto.Bool(false),
				Domain:     proto.String("m.domain"),
			},
			&DnsARecord{
				AddressId:  proto.Int32(0),
				Anonymized: proto.Bool(false),
				Domain:     proto.String("m.domain"),
			},
		},
	}
	traces := map[string]Trace{
		string(lex.EncodeOrDie("node1", "anon1", int64(0), int32(0))): trace,
	}
	consistentRanges := []*store.Record{
		&store.Record{
			Key:   lex.EncodeOrDie("node1", "anon1", int64(0), int32(0)),
			Value: lex.EncodeOrDie("node1", "anon1", int64(0), int32(0)),
		},
	}
	addressIdStore := map[string]string{
		string(lex.EncodeOrDie("node1", "anon1", int64(0), int32(0), int32(0))): string(lex.EncodeOrDie("mac1")),
	}

	runLookupsPerDevicePipeline(traces, consistentRanges, addressIdStore)

	// Output:
	// LookupsPerDevice:
	// node1,mac1,m.domain: 2
	//
	// LookupsPerDevicePerHour:
	// node1,mac1,m.domain,0: 2
}

func ExampleLookupsPerDevice_multipleTraces() {
	trace := Trace{
		ARecord: []*DnsARecord{
			&DnsARecord{
				AddressId:  proto.Int32(0),
				Anonymized: proto.Bool(false),
				Domain:     proto.String("m.domain"),
			},
		},
	}
	traces := map[string]Trace{
		string(lex.EncodeOrDie("node1", "anon1", int64(0), int32(0))):   trace,
		string(lex.EncodeOrDie("node1", "anon1", int64(0), int32(1))):   trace,
		string(lex.EncodeOrDie("node1", "anon1", int64(0), int32(120))): trace,
	}
	consistentRanges := []*store.Record{
		&store.Record{
			Key:   lex.EncodeOrDie("node1", "anon1", int64(0), int32(0)),
			Value: lex.EncodeOrDie("node1", "anon1", int64(0), int32(120)),
		},
	}
	addressIdStore := map[string]string{
		string(lex.EncodeOrDie("node1", "anon1", int64(0), int32(0), int32(0))): string(lex.EncodeOrDie("mac1")),
	}

	runLookupsPerDevicePipeline(traces, consistentRanges, addressIdStore)

	// Output:
	// LookupsPerDevice:
	// node1,mac1,m.domain: 3
	//
	// LookupsPerDevicePerHour:
	// node1,mac1,m.domain,0: 2
	// node1,mac1,m.domain,3600: 1
}

func ExampleLookupsPerDevice_aliasAddresses() {
	trace := Trace{
		ARecord: []*DnsARecord{
			&DnsARecord{
				AddressId:  proto.Int32(0),
				Anonymized: proto.Bool(false),
				Domain:     proto.String("m.domain"),
			},
		},
	}
	traces := map[string]Trace{
		string(lex.EncodeOrDie("node1", "anon1", int64(0), int32(0))): trace,
		string(lex.EncodeOrDie("node1", "anon1", int64(0), int32(1))): trace,
	}
	consistentRanges := []*store.Record{
		&store.Record{
			Key:   lex.EncodeOrDie("node1", "anon1", int64(0), int32(0)),
			Value: lex.EncodeOrDie("node1", "anon1", int64(0), int32(1)),
		},
	}
	addressIdStore := map[string]string{
		string(lex.EncodeOrDie("node1", "anon1", int64(0), int32(0), int32(0))): string(lex.EncodeOrDie("mac1")),
		string(lex.EncodeOrDie("node1", "anon1", int64(0), int32(0), int32(1))): string(lex.EncodeOrDie("mac2")),
	}

	runLookupsPerDevicePipeline(traces, consistentRanges, addressIdStore)

	// Output:
	// LookupsPerDevice:
	// node1,mac1,m.domain: 1
	// node1,mac2,m.domain: 1
	//
	// LookupsPerDevicePerHour:
	// node1,mac1,m.domain,0: 1
	// node1,mac2,m.domain,0: 1
}
