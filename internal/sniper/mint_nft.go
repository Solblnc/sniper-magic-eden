package sniper

import (
	"Sniper-Magic-Eden/internal/models"
	"context"
	"encoding/json"
	"errors"
	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"io"
	"net/http"
	"os"
)

// GetTransactionData - returns a response of a request, then it will be signed
func GetTransactionData(url string) ([]byte, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer"+os.Getenv("ME_APIKEY"))
	if err != nil {
		return nil, err
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		body, _ := io.ReadAll(res.Body)
		return nil, errors.New(string(body))
	}

	var response models.Response
	body, _ := io.ReadAll(res.Body)
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}
	return response.TxSigned.Data, nil
}

// MintNft - checks the data -> signs it -> send a transaction to a cluster
func MintNft(privateKey solana.PrivateKey, url string) (string, error) {
	txSigned, err := GetTransactionData(url)
	if err != nil {
		return "", err
	}

	transaction, err := solana.TransactionFromDecoder(bin.NewBorshDecoder(txSigned))
	if err != nil {
		return "", err
	}

	node := rpc.New(os.Getenv("NODE_ENDPOINT"))

	message, err := transaction.Message.MarshalBinary()
	if err != nil {
		return "", err
	}

	signed, err := privateKey.Sign(message)
	if err != nil {
		return "", err
	}
	transaction.Signatures[0] = signed

	signature, err := node.SendTransactionWithOpts(context.TODO(), transaction, rpc.TransactionOpts{SkipPreflight: true})
	if err != nil {
		return "", err
	}

	return signature.String(), nil

}
