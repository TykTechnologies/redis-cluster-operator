package clustering

import (
	"fmt"

	"github.com/TykTechnologies/redis-cluster-operator/internal/redisutil"
)

// AttachingSlavesToMaster used to attach slaves to their masters
func (c *Ctx) AttachingSlavesToMaster(admin redisutil.IAdmin) error {
	var globalErr error
	for masterID, slaves := range c.slavesByMaster {
		masterNode, err := c.cluster.GetNodeByID(masterID)
		if err != nil {
			c.log.Error(err, fmt.Sprintf("unable fo found the Cluster.Node with redis ID:%s", masterID))
			continue
		}
		for _, slave := range slaves {
			c.log.Info(fmt.Sprintf("attaching node %s to master %s", slave.ID, masterID))

			err := admin.AttachSlaveToMaster(slave, masterNode.ID)
			if err != nil {
				c.log.Error(err, fmt.Sprintf("attaching node %s to master %s", slave.ID, masterID))
				globalErr = err
			}
		}
	}
	return globalErr
}
