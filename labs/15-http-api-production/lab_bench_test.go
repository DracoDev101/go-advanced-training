package lab

import (
    "context"
    "testing"
)

var countSink int64

func BenchmarkProcess(b *testing.B) {
    events := SampleEvents()
    for i:=0; i<b.N; i++ {
        var r Recorder
        if err := Process(context.Background(), &r, events); err != nil { b.Fatal(err) }
        countSink = r.Count()
    }
}
