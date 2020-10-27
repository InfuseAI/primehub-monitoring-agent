package monitoring

import (
	"testing"
	"time"
)

func TestMergeByAverage(t *testing.T) {

	t.Log("Give a buffer with capacity 3")
	{
		buffer := NewBuffer(0, 3)
		for i := 0; i < 3; i++ {
			buffer.Add(Record{
				Timestamp:      time.Now().Unix(),
				CpuUtilization: 10,
				MemoryUsed:     10,
				GPURecords: []GPURecord{
					{
						Index:          0,
						MemoryUsed:     60,
						GPUUtilization: 60,
					},
					{
						Index:          0,
						MemoryUsed:     15,
						GPUUtilization: 15,
					},
				},
			})
		}

		// verify CPU and Memory
		t.Log("Add 3 same records, and merge it by avarage")
		t.Log("cpu: 10, memory: 10, gpu: [{memory: 60, gpu: 60}, {memory: 15, gpu: 15}]")
		record := buffer.LastAverage(3)
		if record.CpuUtilization != 10 {
			t.Error("CpuUtilization should be 10")
		}
		t.Log("CpuUtilization should be 10")

		if record.MemoryUsed != 10 {
			t.Error("MemoryUsed should be 10")
		}
		t.Log("MemoryUsed should be 10")

		if len(record.GPURecords) != 2 {
			t.Error("len(gpus) should be 1")
		}
		t.Log("len(gpus) should be 1")

		if record.GPURecords[0].MemoryUsed != 60 {
			t.Error("MemoryUsed should be 60")
		}
		t.Log("MemoryUsed should be 60")

		if record.GPURecords[0].GPUUtilization != 60 {
			t.Error("GPUUtilization should be 60")
		}
		t.Log("GPUUtilization should be 60")

		if record.GPURecords[1].MemoryUsed != 15 {
			t.Error("MemoryUsed should be 15")
		}
		t.Log("MemoryUsed should be 15")

		if record.GPURecords[1].GPUUtilization != 15 {
			t.Error("GPUUtilization should be 15")
		}
		t.Log("GPUUtilization should be 15")
	}

}
