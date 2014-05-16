package gosteno

type TestingSink struct {
	Records []*Record
}

var theGlobalTestSink *TestingSink

func EnterTestMode(logLevel ...LogLevel) {
	theGlobalTestSink = NewTestingSink()
	stenoConfig := Config{
		Sinks: []Sink{theGlobalTestSink},
	}
	if len(logLevel) > 0 {
		stenoConfig.Level = logLevel[0]
	}
	Init(&stenoConfig)
}

func GetMeTheGlobalTestSink() *TestingSink {
	return theGlobalTestSink
}

func NewTestingSink() *TestingSink {
	return &TestingSink{
		Records: make([]*Record, 0, 10),
	}
}

func (tSink *TestingSink) AddRecord(record *Record) {
	tSink.Records = append(tSink.Records, record)
}

func (tSink *TestingSink) Flush() {

}

func (tSink *TestingSink) SetCodec(codec Codec) {

}

func (tSink *TestingSink) GetCodec() Codec {
	return nil
}
