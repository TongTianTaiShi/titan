package nat

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Filecoin-Titan/titan/api/types"
	"github.com/Filecoin-Titan/titan/node/scheduler/node"
	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("scheduler/nat")

const (
	miniCandidateCount = 2
	detectInterval     = 60
	maxRetry           = 3
)

type Manager struct {
	nodeManager *node.Manager
	retryList   []*retryNode
	lock        *sync.Mutex
	edgeMap     *sync.Map
}

type retryNode struct {
	id    string
	retry int
}

func NewManager(nodeMgr *node.Manager) *Manager {
	m := &Manager{nodeManager: nodeMgr, lock: &sync.Mutex{}, retryList: make([]*retryNode, 0), edgeMap: &sync.Map{}}
	go m.startTicker()

	return m
}

func (m *Manager) startTicker() {
	for {
		time.Sleep(detectInterval * time.Second)

		for len(m.retryList) > 0 {
			node := m.removeNode()
			m.retryDetectNatType(node)
		}
	}

}

func (m *Manager) removeNode() *retryNode {
	m.lock.Lock()
	defer m.lock.Unlock()

	if len(m.retryList) == 0 {
		return nil
	}

	node := m.retryList[0]
	m.retryList = m.retryList[1:]
	return node
}

func (m *Manager) addNode(n *retryNode) {
	m.lock.Lock()
	defer m.lock.Unlock()
	for _, node := range m.retryList {
		if node.id == n.id {
			return
		}
	}

	m.retryList = append(m.retryList, n)
}

func (m *Manager) isInRetryList(nodeID string) bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	for _, node := range m.retryList {
		if node.id == nodeID {
			return true
		}
	}

	return false
}

func (m *Manager) delayDetectNatType(n *retryNode) {
	m.addNode(n)
}

func (m *Manager) retryDetectNatType(node *retryNode) {
	node.retry++
	nodeID := node.id

	_, ok := m.edgeMap.LoadOrStore(nodeID, struct{}{})
	if ok {
		log.Warnf("node %s determining nat type")
		return
	}
	defer m.edgeMap.Delete(nodeID)

	eNode := m.nodeManager.GetNode(nodeID)
	if eNode == nil {
		log.Errorf("node %s offline or not exists", nodeID)
		return
	}

	cNodes := m.nodeManager.GetCandidateNodes(miniCandidateCount)
	natType, err := determineEdgeNATType(context.Background(), eNode, cNodes)
	if err != nil {
		log.Errorf("DetermineNATType error:%s", err.Error())
	}

	eNode.NATType = natType.String()

	if natType == types.NatTypeUnknown && node.retry < maxRetry {
		m.delayDetectNatType(node)
	}
	log.Debugf("retry detect nat type %s", eNode.NATType)
}

func (m *Manager) DetermineEdgeNATType(ctx context.Context, nodeID string) {
	if m.isInRetryList(nodeID) {
		log.Debugf("node %s waiting to retry")
		return
	}

	_, ok := m.edgeMap.LoadOrStore(nodeID, struct{}{})
	if ok {
		log.Warnf("node %s determining nat type")
		return
	}
	defer m.edgeMap.Delete(nodeID)

	eNode := m.nodeManager.GetNode(nodeID)
	if eNode == nil {
		log.Errorf("node %s offline or not exists", nodeID)
		return
	}

	cNodes := m.nodeManager.GetCandidateNodes(miniCandidateCount)

	natType, err := determineEdgeNATType(ctx, eNode, cNodes)
	if err != nil {
		log.Errorf("DetermineNATType error:%s", err.Error())
	}

	eNode.NATType = natType.String()

	if natType == types.NatTypeUnknown {
		m.delayDetectNatType(&retryNode{id: nodeID, retry: 0})
	}
	log.Debugf("nat type %s", eNode.NATType)
}

// GetCandidateURLsForDetectNat Get the rpc url of the specified number of candidate nodes
func (m *Manager) GetCandidateURLsForDetectNat(ctx context.Context) ([]string, error) {
	// minimum of 3 candidates is required for user detect nat
	needCandidateCount := miniCandidateCount + 1
	candidates := m.nodeManager.GetCandidateNodes(needCandidateCount)
	if len(candidates) < needCandidateCount {
		return nil, fmt.Errorf("minimum of %d candidates is required", needCandidateCount)
	}

	urls := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		urls = append(urls, candidate.RPCURL())
	}
	return urls, nil
}
