package stat_consumer

import (
	"fmt"
	"io"

	"github.com/mongodb/mongo-tools/mongostat/stat_consumer/line"
	"github.com/mongodb/mongo-tools/mongostat/status"
)

// StatConsumer maintains the current set of headers and the most recent
// ServerStatus for each host. It creates a StatLine when passed a new record
// and can format and write groups of StatLines.
type StatConsumer struct {
	formatter              LineFormatter
	readerConfig           *status.ReaderConfig
	oldStats               map[string]*status.ServerStatus
	headers, customHeaders []string
	keyNames               map[string]string
	writer                 io.Writer
	flags                  int
}

// NewStatConsumer creates a new StatConsumer with no previous records
func NewStatConsumer(flags int, customHeaders []string, keyNames map[string]string, readerConfig *status.ReaderConfig, formatter LineFormatter, writer io.Writer) (sc *StatConsumer) {
	sc = &StatConsumer{
		formatter:     formatter,
		readerConfig:  readerConfig,
		oldStats:      make(map[string]*status.ServerStatus),
		customHeaders: customHeaders,
		keyNames:      keyNames,
		writer:        writer,
		flags:         flags,
	}
	return sc
}

// Update takes in a ServerStatus and returns a StatLine if it has a previous record
func (sc *StatConsumer) Update(newStat *status.ServerStatus) (l *line.StatLine, seen bool) {
	oldStat, seen := sc.oldStats[newStat.Host]
	sc.oldStats[newStat.Host] = newStat
	if seen {
		l = line.NewStatLine(oldStat, newStat, sc.headers, sc.readerConfig)
		return
	}

	if sc.flags != 0 {
		if status.IsMMAP(newStat) {
			sc.flags |= line.FlagMMAP
		} else if status.IsWT(newStat) {
			sc.flags |= line.FlagWT
		}
		if status.IsReplSet(newStat) {
			sc.flags |= line.FlagRepl
		}
		if status.HasLocks(newStat) {
			sc.flags |= line.FlagLocks
		}

		// Modify headers
		sc.headers = []string{}
		for _, desc := range line.CondHeaders {
			if desc.Flag&sc.flags == desc.Flag {
				sc.headers = append(sc.headers, desc.Key)
			}
		}
		sc.headers = append(sc.headers, sc.customHeaders...)
	}
	return
}

// FormatLines consumes StatLines, formats them, and sends them to its writer
func (sc *StatConsumer) FormatLines(lines []*line.StatLine) {
	str := sc.formatter.FormatLines(lines, sc.headers, sc.keyNames)
	fmt.Fprintf(sc.writer, "%s", str)
}
