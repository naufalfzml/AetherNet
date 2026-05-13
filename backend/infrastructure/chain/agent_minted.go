package chain

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
)

const AgentMintedTopic = "0xaa2cc66148b7179c55c85fe5e7e8d910799c8b60830eb01df80d973aa8ac8764"

type AgentMintedEvent struct {
	TokenID         string
	OwnerAddress    string
	MetadataPointer string
	AgentAddress    string
	BlockNumber     *big.Int
}

type LogScanner struct {
	RPCURL     string
	HTTPClient *http.Client
}

func (s LogScanner) Health(ctx context.Context) error {
	_, err := s.LatestBlock(ctx)
	return err
}

func (s LogScanner) LatestBlock(ctx context.Context) (*big.Int, error) {
	var result string
	if err := s.rpc(ctx, "eth_blockNumber", []any{}, &result); err != nil {
		return nil, err
	}
	return hexToBig(result)
}

func (s LogScanner) AgentMintedEvents(ctx context.Context, registry string, fromBlock *big.Int, toBlock *big.Int) ([]AgentMintedEvent, error) {
	var logs []rpcLog
	params := []any{map[string]any{
		"address":   registry,
		"fromBlock": bigToHex(fromBlock),
		"toBlock":   bigToHex(toBlock),
		"topics":    []any{AgentMintedTopic},
	}}
	if err := s.rpc(ctx, "eth_getLogs", params, &logs); err != nil {
		return nil, err
	}

	events := make([]AgentMintedEvent, 0, len(logs))
	for _, log := range logs {
		event, err := DecodeAgentMintedLog(log.Topics, log.Data, log.BlockNumber)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

func (s LogScanner) rpc(ctx context.Context, method string, params []any, result any) error {
	client := s.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	body, err := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.RPCURL, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("rpc status %d", res.StatusCode)
	}
	var envelope rpcResponse
	if err := json.NewDecoder(res.Body).Decode(&envelope); err != nil {
		return err
	}
	if envelope.Error != nil {
		return fmt.Errorf("rpc %s: %s", method, envelope.Error.Message)
	}
	if envelope.Result == nil {
		return errors.New("rpc missing result")
	}
	return json.Unmarshal(envelope.Result, result)
}

func DecodeAgentMintedLog(topics []string, data string, blockNumber string) (AgentMintedEvent, error) {
	if len(topics) < 3 {
		return AgentMintedEvent{}, errors.New("agent minted log missing indexed topics")
	}
	tokenID, err := topicBig(topics[1])
	if err != nil {
		return AgentMintedEvent{}, err
	}
	owner, err := topicAddress(topics[2])
	if err != nil {
		return AgentMintedEvent{}, err
	}
	metadataPointer, agentAddress, err := decodeAgentMintedData(data)
	if err != nil {
		return AgentMintedEvent{}, err
	}
	block, err := hexToBig(blockNumber)
	if err != nil {
		return AgentMintedEvent{}, err
	}
	return AgentMintedEvent{
		TokenID:         tokenID.String(),
		OwnerAddress:    owner,
		MetadataPointer: metadataPointer,
		AgentAddress:    agentAddress,
		BlockNumber:     block,
	}, nil
}

func decodeAgentMintedData(data string) (string, string, error) {
	decoded, err := hex.DecodeString(strings.TrimPrefix(data, "0x"))
	if err != nil {
		return "", "", err
	}
	if len(decoded) < 96 {
		return "", "", errors.New("agent minted data too short")
	}
	offset := int(new(big.Int).SetBytes(decoded[0:32]).Int64())
	agentAddress := "0x" + hex.EncodeToString(decoded[44:64])
	if offset < 64 || offset+32 > len(decoded) {
		return "", "", errors.New("agent minted string offset out of range")
	}
	length := int(new(big.Int).SetBytes(decoded[offset : offset+32]).Int64())
	start := offset + 32
	end := start + length
	if length < 0 || end > len(decoded) {
		return "", "", errors.New("agent minted string length out of range")
	}
	return string(decoded[start:end]), agentAddress, nil
}

func topicBig(topic string) (*big.Int, error) {
	value, err := hexToBig(topic)
	if err != nil {
		return nil, err
	}
	return value, nil
}

func topicAddress(topic string) (string, error) {
	decoded, err := hex.DecodeString(strings.TrimPrefix(topic, "0x"))
	if err != nil {
		return "", err
	}
	if len(decoded) != 32 {
		return "", errors.New("topic address must be 32 bytes")
	}
	return "0x" + hex.EncodeToString(decoded[12:32]), nil
}

func hexToBig(value string) (*big.Int, error) {
	parsed, ok := new(big.Int).SetString(strings.TrimPrefix(value, "0x"), 16)
	if !ok {
		return nil, fmt.Errorf("invalid hex integer %q", value)
	}
	return parsed, nil
}

func bigToHex(value *big.Int) string {
	if value == nil {
		return "0x0"
	}
	return "0x" + value.Text(16)
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcLog struct {
	Topics      []string `json:"topics"`
	Data        string   `json:"data"`
	BlockNumber string   `json:"blockNumber"`
}
