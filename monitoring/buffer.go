package monitoring

import "time"

type Buffer struct {
	Data          []Record
	NextIndex     int64
	Max           int
	LastUpdated   time.Time
	Interval      int
	AverageByLast int
}

func (b *Buffer) Add(record Record) {
	b.Data[b.NextIndex%int64(b.Max)] = record
	b.NextIndex++
	b.LastUpdated = time.Now()
}

func (b *Buffer) Last(request int) []Record {
	last := request
	if int64(last) > b.NextIndex {
		last = int(b.NextIndex)
	}

	p := make([]Record, last)
	fromIndex := b.NextIndex - int64(last)
	for i := fromIndex; i < fromIndex+int64(last); i++ {
		p[i-fromIndex] = b.Data[i%int64(b.Max)]
	}

	return p
}

func (b *Buffer) LastAvailable() []Record {
	return b.Last(b.Max)
}

func (b *Buffer) IsTimeToUpdate() bool {
	if b.NextIndex == 0 {
		return true
	}

	return b.LastUpdated.Before(time.Now().Add(time.Duration(-b.Interval) * time.Second))
}

func (b *Buffer) HasLast(last int) bool {
	return b.NextIndex >= int64(last) && last <= b.Max
}

func (b *Buffer) LastAverage(request int) Record {
	last := request
	if int64(last) > b.NextIndex {
		last = int(b.NextIndex)
	}

	var record = Record{
		Timestamp:      0,
		CpuUtilization: 0,
		MemoryUsed:     0,
		GPURecords:     make([]GPURecord, 0),
	}

	fromIndex := b.NextIndex - int64(last)
	for i := fromIndex; i < fromIndex+int64(last); i++ {
		r := b.Data[int(i%int64(b.Max))]

		// init record
		if i == fromIndex {
			record.Timestamp = r.Timestamp
			if r.GPURecords != nil && len(r.GPURecords) != 0 {
				record.GPURecords = make([]GPURecord, len(r.GPURecords))
			}
		}
		//sum += b.Buffer[int(i%b.Max)].Value
		record.CpuUtilization += r.CpuUtilization
		record.MemoryUsed += r.MemoryUsed

		if record.GPURecords != nil {
			for g := 0; g < len(r.GPURecords); g++ {
				record.GPURecords[g].Index = r.GPURecords[g].Index
				record.GPURecords[g].GPUUtilization += r.GPURecords[g].GPUUtilization
				record.GPURecords[g].MemoryUsed += r.GPURecords[g].MemoryUsed
			}
		}
	}

	record.CpuUtilization /= last
	record.MemoryUsed /= int64(last)
	if record.GPURecords != nil {
		for g := 0; g < len(record.GPURecords); g++ {
			record.GPURecords[g].GPUUtilization /= last
			record.GPURecords[g].MemoryUsed /= int64(last)
		}
	}

	return record
}

func NewBuffer(interval int, size int) *Buffer {
	p := new(Buffer)
	p.Max = size
	p.NextIndex = 0
	p.Data = make([]Record, size)
	p.Interval = interval
	return p
}
