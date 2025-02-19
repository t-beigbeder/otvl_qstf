package gst

import (
	"fmt"
	_ "github.com/reugn/go-streams"
	ext "github.com/reugn/go-streams/extension"
	"github.com/reugn/go-streams/flow"
	"github.com/t-beigbeder/otvl_qstf/internal/gst"
)

type DataSample struct {
	Title string   `yaml:"title"`
	Words int      `yaml:"words"`
	Tags  []string `yaml:"tags"`
	Type  string   `yaml:"type"`
}

func dataSet() *gst.YamlSource[DataSample] {
	return gst.NewYamlSource("examples/gst/testdata/mfw_sample.yaml", func() []DataSample {
		return []DataSample{}
	})
}

func DisplayDataSet() {
	tsf := func(v any) string {
		return fmt.Sprintf("%v", v)
	}
	source := dataSet()
	tsFlow := flow.NewMap(tsf, 1)
	sink := ext.NewStdoutSink()
	source.Via(tsFlow).To(sink)
}

func TotalWordCount() {
	ewc := func(entry DataSample) int {
		return entry.Words
	}
	sumWc := func(cwc, redWc int) int {
		return cwc + redWc
	}
	source := dataSet()
	ewcFlow := flow.NewMap(ewc, 10)
	sumFlow := flow.NewReduce(sumWc)
	sink := ext.NewStdoutSink()
	source.Via(ewcFlow).Via(sumFlow).To(sink)
}
