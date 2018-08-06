package member

import (
	"github.com/google/uuid"
	"time"
	"fmt"
	"math/rand"
	"github.com/ipfs/go-ipld-cbor"
	"avalanche/storage"
	// crytpoRand "crypto/rand"
)

type NodeId string

type NodeSystem struct {
	N        int
	K        int
	Alpha    int // slight deviation to avoid floats, just calculate k*a from the paper
	BetaOne  int
	BetaTwo int
	Nodes    NodeHolder
	Metadata map[string]interface{}
}

type transactionQuery struct {
	transaction *cbornode.Node
	responseChan chan *cbornode.Node
}

type Node struct {
	Id NodeId
	State *cbornode.Node
	LastState *cbornode.Node
	Incoming chan transactionQuery
	StopChan chan bool
	Counts map[*cbornode.Node]int
	Count int
	System *NodeSystem
	Accepted bool
	OnQuery func(*Node, *cbornode.Node, chan *cbornode.Node)
	OnStart func(*Node)
	OnStop func(*Node)
	Metadata map[string]interface{}
	Storage storage.Storage
}

type NodeHolder map[NodeId]*Node

func NewNode(system *NodeSystem, storage storage.Storage) *Node {
	return &Node{
		Id: NodeId(uuid.New().String()),
		Incoming: make(chan transactionQuery, system.BetaOne),
		StopChan: make(chan bool),
		Counts: make(map[*cbornode.Node]int),
		System: system,
		Metadata: make(map[string]interface{}),
		Storage: storage,
	}
}

func (n *Node) Start() error {
	go func() {
		for {
			select {
			case <-n.StopChan:
				break
			case query := <- n.Incoming:
				n.OnQuery(n, query.transaction, query.responseChan)
			}
		}
	}()
	if n.OnStart != nil {
		n.OnStart(n)
	}
	return nil
}

func (n *Node) Stop() error {
	n.StopChan <- true
	if n.OnStop != nil {
		n.OnStop(n)
	}
	return nil
}


func (n *Node) SendQuery(state *cbornode.Node) (*cbornode.Node,error) {
	t := time.After(10 * time.Second)
	respChan := make(chan *cbornode.Node)
	n.Incoming <- transactionQuery{
		transaction: state,
		responseChan: respChan,
	}
	select {
	case <-t:
		fmt.Printf("timeout on sendquery")
		return nil, fmt.Errorf("timeout")
	case resp := <-respChan:
		return resp,nil
	}
}

//TODO: this is not cryptographically sound
func (nh NodeHolder) RandNode() *Node {
	i := rand.Intn(len(nh))
	for _,node := range nh {
		if i == 0 {
			return node
		}
		i--
	}
	panic("never")
}

func makeRange(min, max int) []int {
    a := make([]int, max-min+1)
    for i := range a {
        a[i] = min + i
    }
    return a
}

func shuffle(slice []int) {
   r := rand.New(rand.NewSource(time.Now().Unix()))
   for n := len(slice); n > 0; n-- {
      randIndex := r.Intn(n)
      slice[n-1], slice[randIndex] = slice[randIndex], slice[n-1]
   }
}

func intInSlice(num int, list []int) bool {
 for _, v := range list {
	 if v == num {
		 return true
	 }
 }
 return false
}

func (nh NodeHolder) RandKOfNNodes(k int, n int) []*Node {
	nodes := make([]*Node, k)

	nodeIndicesPreShuffle := makeRange(0, n - 1)
	shuffle(nodeIndicesPreShuffle)

	nodeIndices := make([]int, k)

	for i := 0; i < k; i++ {
		nodeIndices[i] = nodeIndicesPreShuffle[i]
	}

	a := 0
	i := n
	for _,node := range nh {
		if intInSlice(i, nodeIndices) {
			nodes[a] = node
			a++
		}
		i--
	}

	return nodes
}
