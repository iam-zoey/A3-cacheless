package shared

// import (
// 	"fmt"
// 	"log"
// 	"time"
// )

// const (
// 	HeartbeatInterval = 3  // Seconds between heartbeat messages
// 	HeartbeatTimeout  = 10 // Seconds to wait for a response
// )

// type HeartbeatMessage struct {
// 	From   string
// 	Status string // "alive" or "failed"
// }

// type FailoverMessage struct {
// 	NewTraderID string
// }

// // StartHeartbeat starts the periodic heartbeat protocol.
// func (n *Node) StartHeartbeat() {
// 	t := n.Role.(*Trader)
// 	ticker := time.NewTicker(HeartbeatInterval * time.Second)
// 	defer ticker.Stop()

// 	for range ticker.C {
// 		if n.IsActive {
// 			go n.SendHeartbeat()
// 		}
// 	}
// }

// // SendHeartbeat sends a heartbeat to the peer trader.
// func (n *Node) SendHeartbeat() {
// 	t := n.Role.(*Trader)
// 	peerID, err := String2Int(t.Peer)
// 	if err != nil {
// 		fmt.Println("Error converting peer ID to string")
// 	}
// 	client, err := GetClient(peerID)
// 	if err != nil {
// 		log.Printf("Trader %s: Peer %s not responding. Assuming failure.", n.ID, t.Peer)
// 		n.HandlePeerFailure()
// 		return
// 	}
// 	defer client.Close()

// 	var reply HeartbeatMessage
// 	ID := Int2String(n.ID)
// 	if err != nil {
// 		fmt.Println("Error converting node ID to string")
// 	}

// 	req := HeartbeatMessage{
// 		From:   ID,
// 		Status: "alive",
// 	}

// 	err = client.Call(fmt.Sprintf("Node%s.ReceiveHeartbeat", t.Peer), &req, &reply)
// 	if err != nil {
// 		log.Printf("Trader %s: Failed to send heartbeat to %s: %v", n.ID, t.Peer, err)
// 		n.HandlePeerFailure()
// 	}
// }

// // ReceiveHeartbeat handles incoming heartbeat messages.
// func (n *Node) ReceiveHeartbeat(req *HeartbeatMessage, reply *HeartbeatMessage) error {
// 	t := n.Role.(*Trader)
// 	t.HeartbeatMu.Lock()
// 	t.Heartbeat = true
// 	t.HeartbeatMu.Unlock()

// 	log.Printf("Trader %s: Received heartbeat from %s", t.ID, req.From)
// 	reply.From = n.ID
// 	reply.Status = "alive"
// 	return nil
// }

// // HandlePeerFailure handles the failover when the peer trader is unresponsive.
// func (n *Node) HandlePeerFailure() {
// 	t := n.Role.(*Trader)
// 	t.HeartbeatMu.Lock()
// 	defer t.HeartbeatMu.Unlock()

// 	if !n.IsActive {
// 		return
// 	}

// 	log.Printf("Trader %s: Peer %s is assumed to have failed. Becoming the sole trader.", t.ID, t.PeerTrader)
// 	n.NotifyPeersOfFailover()
// }

// // NotifyPeersOfFailover notifies all peers of the new active trader.
// func (n *Node) NotifyPeersOfFailover() {
// 	t := n.Role.(*Trader)
// 	go func(peer string) {
// 		PeerID, err := String2Int(peer)
// 		client, err := GetClient(PeerID)
// 		if err != nil {
// 			log.Printf("Trader %s: Failed to notify peer %s: %v", n.ID, t.peer, err)
// 			return
// 		}
// 		defer client.Close()

// 		req := FailoverMessage{NewTraderID: n.ID}
// 		var reply Message

// 		err = client.Call("Node.HandleFailover", &req, &reply)
// 		if err != nil {
// 			log.Printf("Trader %s: Failed to notify peer %s: %v", n.ID, peer, err)
// 		}
// 	}(peer)
// }

// func (n *Node) DeactivateTrader() {
// 	n.IsActive = false
// 	return
// }
