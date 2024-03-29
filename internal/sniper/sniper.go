package sniper

import (
	"context"
	"github.com/blocto/solana-go-sdk/client"
	"github.com/blocto/solana-go-sdk/rpc"
	"github.com/gagliardetto/solana-go"
	"log"
	"math/rand"
	"os"
	"sniper/internal/models"
	"sniper/internal/utils"
	"strconv"
	"strings"
	"time"

	"github.com/gagliardetto/solana-go/rpc/ws"
)

var (
	MEPublicKeyStr = "1BWutmTvYPwDtmw9abTkS4Ssr8no61spGAvW1X6NDix"
	MEPublicKey    = solana.MustPublicKeyFromBase58(MEPublicKeyStr)
)

type Sniper struct {
	cli         *client.Client
	actions     chan *models.Token
	collections map[string]*models.Token
	privateKey  solana.PrivateKey
}

type Options struct {
	Endpoint   string
	Actions    chan *models.Token
	PrivateKey string
}

func New(options *Options) (*Sniper, error) {
	collections, err := utils.LoadCollections()
	if err != nil {
		return nil, err
	}

	cli := client.NewClient(options.Endpoint)
	// node health check
	if _, err = cli.GetBalance(context.Background(), MEPublicKeyStr); err != nil {
		return nil, err
	}

	var privateKey solana.PrivateKey
	if options.PrivateKey != "" {
		privateKey, err = solana.PrivateKeyFromBase58(options.PrivateKey)
		if err != nil {
			return nil, err
		}
	}

	return &Sniper{
		cli:         cli,
		actions:     options.Actions,
		collections: collections,
		privateKey:  privateKey,
	}, nil
}

func (s *Sniper) Start(ctx context.Context) error {
	client, err := ws.Connect(context.Background(), rpc.MainNetBeta_WS)
	if err != nil {
		return err
	}

	sub, err := client.LogsSubscribeMentions(MEPublicKey, "confirmed")
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()

	go func() {
		for {
			got, err := sub.Recv()
			if err != nil {
				log.Println(err)
				return
			}

			if got == nil {
				continue
			}

			go s.GetTransaction(ctx, got.Value.Signature.String())
		}
	}()

	<-ctx.Done()
	return nil
}

func (s *Sniper) GetTransaction(ctx context.Context, signature string) {
	var (
		transaction *client.Transaction
		err         error
	)
	// Sleep until transaction data can be obtained
	time.Sleep(time.Millisecond*time.Duration(rand.Intn(1000)) + 500)
	for transaction == nil {
		transaction, err = s.cli.GetTransactionWithConfig(
			ctx,
			signature,
			rpc.GetTransactionConfig{Commitment: "confirmed"},
		)
		if err != nil || transaction == nil {
			time.Sleep(time.Millisecond * 500)
			continue
		}
	}

	token := s.parseTransaction(transaction)
	if token == nil {
		return
	}

	// set floor price of the collection
	token.FloorPrice = GetFloor(token.Symbol)
	s.actions <- token

	if token.Type == "buy" || os.Getenv("ME_APIKEY") == "" || s.privateKey == nil {
		return
	}

	// auto buy conditions
	if token.Price < 0.1 {
		signature, err := s.BuyNFT(token)
		if err != nil {
			log.Println("Error while buying nft:", err.Error())
			return
		}
		log.Println("Signature -", signature)
		log.Println("Successfully bought item.")
	}
}

func (s *Sniper) parseTransaction(transaction *client.GetTransactionResponse) *models.Token {
	var (
		token *models.Token
		ok    bool
	)

	preTokenBalances := transaction.Meta.PreTokenBalances
	postTokenBalances := transaction.Meta.PostTokenBalances
	if len(preTokenBalances) == 0 {
		return nil
	}

	// Check if collections.json contains token
	mintAddress := preTokenBalances[0].Mint
	if token, ok = s.collections[mintAddress]; !ok {
		return nil
	}

	price := getActionPrice(transaction.Meta.LogMessages)
	if price == 0 {
		return nil
	}

	actionType := getActionType(preTokenBalances[0].Owner, postTokenBalances[0].Owner)
	if actionType == "" {
		return nil
	}

	token.Type = actionType
	token.Timestamp = time.Now().UTC().Unix()
	token.BlockTimestamp = *transaction.BlockTime
	token.MintAddress = mintAddress
	token.Price = price
	token.TokenAddress = transaction.Transaction.Message.Accounts[2].String()
	token.Seller = transaction.Transaction.Message.Accounts[0].String()

	return token

}

func getActionPrice(logs []string) float64 {
	var price float64
	for _, msg := range logs {
		if strings.Contains(msg, "price") {
			if len(strings.Split(msg, "price\":")) < 2 {
				return 0
			}

			priceStr := strings.Split(strings.Split(msg, "price\":")[1], ",")[0]
			price, _ = strconv.ParseFloat(priceStr, 64)
			price /= 1e9
		}
	}

	return price
}

func getActionType(preTokenOwner, postTokenOwner string) string {
	if preTokenOwner == MEPublicKeyStr && postTokenOwner != MEPublicKeyStr {
		return "buy"
	} else if postTokenOwner == MEPublicKeyStr {
		return "list"
	}

	return ""
}
