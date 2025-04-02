package clustering

import (
	"testing"

	"github.com/TykTechnologies/redis-cluster-operator/internal/redisutil"
)

func Test_computeReshardTable(t *testing.T) {
	type args struct {
		src      redisutil.Nodes
		numSlots int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "",
			args: args{
				src: redisutil.Nodes{&redisutil.Node{
					ID:    "node1",
					IP:    "10.1.1.1",
					Port:  "6379",
					Role:  "master",
					Slots: redisutil.BuildSlotSlice(5461, 10922),
				}},
				numSlots: 1366,
			},
			want: 1366,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := computeReshardTable(tt.args.src, tt.args.numSlots); len(got) != tt.want {
				t.Errorf("computeReshardTable() = %v, want %v", len(got), tt.want)
			}
		})
	}
}
