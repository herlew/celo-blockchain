package backend

import (
	"net"
	"testing"

	"github.com/ethereum/go-ethereum/consensus/consensustest"
	"github.com/ethereum/go-ethereum/consensus/istanbul/backend/internal/enodes"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

func TestHandleIstAnnounce(t *testing.T) {
	_, b := newBlockChain(4, true)
	for b == nil || b.Address() == getAddress() {
		_, b = newBlockChain(4, true)
	}

	block := b.currentBlock()
	valSet := b.getValidators(block.Number().Uint64(), block.Hash())

	val1PrivateKey, _ := generatePrivateKey()
	val1IPAddress := net.ParseIP("1.2.3.4")
	val1Node := enode.NewV4(&val1PrivateKey.PublicKey, val1IPAddress, 0, 0)
	val1Addr := getAddress()
	val1P2pServer := &consensustest.MockP2PServer{Node: val1Node}

	// Set backend to val1
	b.SetP2PServer(val1P2pServer)
	b.Authorize(val1Addr, &val1PrivateKey.PublicKey, decryptFn, signerFn, signerBLSHashFn, signerBLSMessageFn)

	val2Address := valSet.GetByIndex(2).Address()

	sav, err := b.generateSignedAnnounceVersion(1)
	if err != nil {
		t.Fatalf("Error generating signed announce version: %v", err)
	}
	// Pretend the signed announce version belongs to a different address
	sav.Address = val2Address
	_, err = b.signedAnnounceVersionTable.Upsert([]*enodes.SignedAnnounceVersionEntry{sav.Entry()})
	if err != nil {
		t.Fatalf("Error upserting signed announce version: %v", err)
	}

	// Generate an ist announce message using val1
	istMsg, err := b.generateAnnounce(getTimestamp())
	if err != nil {
		t.Fatalf("Error on generateAnnounce: %s", err)
	}
	istMsg.Sign(b.Sign)
	payload, _ := istMsg.Payload()

	// Set backend to val2
	b.address = val2Address

	// Handle val1's announce message
	if err = b.handleAnnounceMsg(nil, payload); err != nil {
		t.Errorf("error %v", err)
	}

	if node, err := b.valEnodeTable.GetNodeFromAddress(val1Addr); err == nil {
		if node == nil || node.String() != val1Node.String() {
			t.Errorf("Expected %v, but got %v instead", val1Node.String(), node)
		}
	} else {
		t.Errorf("Failed to save enode entry")
	}
}
