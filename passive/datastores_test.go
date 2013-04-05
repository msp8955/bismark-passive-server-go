package passive

import (
	"fmt"
	"github.com/sburnett/transformer"
	"github.com/sburnett/transformer/key"
)

func makeRecordToInclude(nodeId string, sequenceNumber int32) *transformer.LevelDbRecord {
	traceKey := TraceKey{
		NodeId:               []byte(nodeId),
		AnonymizationContext: []byte("context"),
		SessionId:            0,
		SequenceNumber:       sequenceNumber,
	}
	return &transformer.LevelDbRecord{
		Key:   key.EncodeOrDie(&traceKey),
		Value: []byte{},
	}
}

func runIncludeNodes(records transformer.StoreReader) {
	records.BeginReading()
	for {
		record, err := records.ReadRecord()
		if err != nil {
			panic(err)
		}
		if record == nil {
			break
		}
		var traceKey TraceKey
		key.DecodeOrDie(record.Key, &traceKey)
		fmt.Printf("%s %d\n", traceKey.NodeId, traceKey.SequenceNumber)
	}
	records.EndReading()
}

func ExampleIncludeNodes() {
	inputRecords := transformer.SliceStore{}
	inputRecords.BeginWriting()
	inputRecords.WriteRecord(makeRecordToInclude("node1", 1))
	inputRecords.WriteRecord(makeRecordToInclude("node1", 2))
	inputRecords.WriteRecord(makeRecordToInclude("node2", 3))
	inputRecords.WriteRecord(makeRecordToInclude("node2", 4))
	inputRecords.WriteRecord(makeRecordToInclude("node2", 5))
	inputRecords.WriteRecord(makeRecordToInclude("node3", 6))
	inputRecords.EndWriting()

	outputRecords := IncludeNodes(&inputRecords, "node2")
	runIncludeNodes(outputRecords)

	// Output:
	// node2 3
	// node2 4
	// node2 5
}

func ExampleIncludeNodes_multipleNodes() {
	inputRecords := transformer.SliceStore{}
	inputRecords.BeginWriting()
	inputRecords.WriteRecord(makeRecordToInclude("node1", 1))
	inputRecords.WriteRecord(makeRecordToInclude("node1", 2))
	inputRecords.WriteRecord(makeRecordToInclude("node2", 3))
	inputRecords.WriteRecord(makeRecordToInclude("node2", 4))
	inputRecords.WriteRecord(makeRecordToInclude("node2", 5))
	inputRecords.WriteRecord(makeRecordToInclude("node3", 6))
	inputRecords.WriteRecord(makeRecordToInclude("node3", 7))
	inputRecords.WriteRecord(makeRecordToInclude("node4", 8))
	inputRecords.WriteRecord(makeRecordToInclude("node4", 9))
	inputRecords.WriteRecord(makeRecordToInclude("node5", 10))
	inputRecords.WriteRecord(makeRecordToInclude("node6", 11))
	inputRecords.WriteRecord(makeRecordToInclude("node7", 12))
	inputRecords.EndWriting()

	outputRecords := IncludeNodes(&inputRecords, "node2", "node4", "node5", "node7")
	runIncludeNodes(outputRecords)

	// Output:
	// node2 3
	// node2 4
	// node2 5
	// node4 8
	// node4 9
	// node5 10
	// node7 12
}
