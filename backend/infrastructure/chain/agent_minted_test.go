package chain

import (
	"encoding/hex"
	"strings"
	"testing"
)

func TestDecodeAgentMintedLog(t *testing.T) {
	topics := []string{
		AgentMintedTopic,
		wordHex("1"),
		addressTopic("0x1111111111111111111111111111111111111111"),
	}
	data := encodeAgentMintedData("stub://agent-1", "0x6f1330f207ab5e2a52c550af308ba28e3c517311")

	event, err := DecodeAgentMintedLog(topics, data, "0x7b")
	if err != nil {
		t.Fatal(err)
	}
	if event.TokenID != "1" {
		t.Fatalf("expected token id 1, got %s", event.TokenID)
	}
	if event.OwnerAddress != "0x1111111111111111111111111111111111111111" {
		t.Fatalf("unexpected owner %s", event.OwnerAddress)
	}
	if event.MetadataPointer != "stub://agent-1" {
		t.Fatalf("unexpected metadata pointer %s", event.MetadataPointer)
	}
	if event.AgentAddress != "0x6f1330f207ab5e2a52c550af308ba28e3c517311" {
		t.Fatalf("unexpected agent address %s", event.AgentAddress)
	}
	if event.BlockNumber.String() != "123" {
		t.Fatalf("unexpected block %s", event.BlockNumber)
	}
}

func encodeAgentMintedData(metadataPointer string, agentAddress string) string {
	metadataBytes := []byte(metadataPointer)
	padding := 32 - (len(metadataBytes) % 32)
	if padding == 32 {
		padding = 0
	}
	data := strings.Builder{}
	data.WriteString(strings.Repeat("0", 62))
	data.WriteString("40")
	data.WriteString(strings.Repeat("0", 24))
	data.WriteString(strings.TrimPrefix(strings.ToLower(agentAddress), "0x"))
	data.WriteString(wordHexLength(len(metadataBytes)))
	data.WriteString(hex.EncodeToString(metadataBytes))
	data.WriteString(strings.Repeat("0", padding*2))
	return "0x" + data.String()
}

func wordHex(value string) string {
	return "0x" + strings.Repeat("0", 63) + value
}

func wordHexLength(length int) string {
	hexLength := strings.TrimPrefix(hex.EncodeToString([]byte{byte(length)}), "0x")
	return strings.Repeat("0", 64-len(hexLength)) + hexLength
}

func addressTopic(address string) string {
	return "0x" + strings.Repeat("0", 24) + strings.TrimPrefix(strings.ToLower(address), "0x")
}
