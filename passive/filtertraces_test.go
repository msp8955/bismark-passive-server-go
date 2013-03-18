package passive

import (
	"fmt"
	"github.com/sburnett/transformer"
)

func makeSessionRecord(nodeId string, sessionId int64, sequenceNumber int32) *transformer.LevelDbRecord {
	traceKey := TraceKey{
		NodeId:               []byte(nodeId),
		AnonymizationContext: []byte("context"),
		SessionId:            sessionId,
		SequenceNumber:       sequenceNumber,
	}
	return &transformer.LevelDbRecord{
		Key:   EncodeTraceKey(&traceKey),
		Value: []byte{},
	}
}

func makeRangeRecord(nodeId string, sessionId int64, firstSequenceNumber, lastSequenceNumber int32) *transformer.LevelDbRecord {
	traceKey := TraceKey{
		NodeId:               []byte(nodeId),
		AnonymizationContext: []byte("context"),
		SessionId:            sessionId,
		SequenceNumber:       firstSequenceNumber,
	}
	traceValue := TraceKey{
		NodeId:               []byte(nodeId),
		AnonymizationContext: []byte("context"),
		SessionId:            sessionId,
		SequenceNumber:       lastSequenceNumber,
	}
	return &transformer.LevelDbRecord{
		Key:   EncodeTraceKey(&traceKey),
		Value: EncodeTraceKey(&traceValue),
	}
}

func runFilterSessionsPipeline(startSecs, endSecs int64, tracesStore, traceKeyRangesStore, filteredStore *transformer.SliceStore) {
	transformer.RunPipeline(FilterSessionsPipeline(startSecs, endSecs, tracesStore, traceKeyRangesStore, filteredStore), 0)

	filteredStore.BeginReading()
	for {
		record, err := filteredStore.ReadRecord()
		if err != nil {
			panic(err)
		}
		if record == nil {
			break
		}
		traceKey := DecodeTraceKey(record.Key)
		fmt.Printf("%s %d %d\n", traceKey.NodeId, traceKey.SessionId, traceKey.SequenceNumber)
	}
	filteredStore.EndReading()
}

func ExampleFilterSessions() {
	usecs := int64(1000000)

	traceKeyRangesStore := transformer.SliceStore{}
	traceKeyRangesStore.BeginWriting()
	traceKeyRangesStore.WriteRecord(makeRangeRecord("node", 30*usecs, 0, 2))
	traceKeyRangesStore.WriteRecord(makeRangeRecord("node", 31*usecs, 0, 1))
	traceKeyRangesStore.WriteRecord(makeRangeRecord("node", 100*usecs, 0, 10))
	traceKeyRangesStore.WriteRecord(makeRangeRecord("node", 200*usecs, 2, 8))
	traceKeyRangesStore.BeginWriting()

	tracesStore := transformer.SliceStore{}
	tracesStore.BeginWriting()
	tracesStore.WriteRecord(makeSessionRecord("node", 30*usecs, 1))
	tracesStore.WriteRecord(makeSessionRecord("node", 31*usecs, 3))
	tracesStore.WriteRecord(makeSessionRecord("node", 100*usecs, 2))
	tracesStore.WriteRecord(makeSessionRecord("node", 200*usecs, 3))
	tracesStore.EndWriting()

	filteredStore := transformer.SliceStore{}

	runFilterSessionsPipeline(80, 120, &tracesStore, &traceKeyRangesStore, &filteredStore)

	// Output:
	// node 30000000 1
	// node 100000000 2
}