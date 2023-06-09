package sniper

import (
	collections "Sniper-Magic-Eden/internal/collections"
	"Sniper-Magic-Eden/internal/models"
	"context"
	"fmt"
	"github.com/gagliardetto/solana-go"

	rpc_ "github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
	client2 "github.com/portto/solana-go-sdk/client"
	"github.com/portto/solana-go-sdk/rpc"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	MEPublicKeyStr = "1BWutmTvYPwDtmw9abTkS4Ssr8no61spGAvW1X6NDix"
	MEPublicKey    = solana.MustPublicKeyFromBase58(MEPublicKeyStr)
)

type Sniper struct {
	privateKey  solana.PrivateKey
	action      chan *models.Token
	collections map[string]*models.Token
	cli         *client2.Client
}

type Options struct {
	EndPoint   string
	PrivateKey string
	Actions    chan *models.Token
}

// New - returns a new sniper
func New(options *Options) (*Sniper, error) {
	collection, err := collections.LoadCollections()
	if err != nil {
		return nil, err
	}

	cli := client2.NewClient(options.EndPoint)

	// check node health
	if _, err = cli.GetBalance(context.Background(), MEPublicKeyStr); err != nil {
		return nil, err
	}

	return &Sniper{
		privateKey:  solana.PrivateKey{},
		action:      options.Actions,
		collections: collection,
		cli:         cli,
	}, nil
}

// Start - starts a sniper service
func (s *Sniper) Start() error {
	// Start to listen websocket
	client, err := ws.Connect(context.Background(), rpc_.MainNetBeta_WS)
	if err != nil {
		return err
	}

	sub, err := client.LogsSubscribeMentions(MEPublicKey, "confirmed")
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()

	for {
		got, err := sub.Recv()
		if err != nil {
			return err
		}
		go s.GetTransaction(context.Background(), got.Value.Signature.String())
	}

}

func (s *Sniper) GetTransaction(ctx context.Context, signature string) {
	var (
		transaction *client2.GetTransactionResponse
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
	if token != nil {
		// set floor price of the collection
		token.FloorPrice = GetFloor(token.Symbol)
		s.action <- token

		if token.Type == "buy" || os.Getenv("ME_APIKEY") == "" || s.privateKey == nil {
			return
		}

		// auto buy conditions
		if token.Price < 0.1 {
			buyURL := fmt.Sprintf(`https://api-mainnet.magiceden.dev/v2/instructions/buy_now?buyer=%s&seller=%s&auctionHouseAddress=E8cU1WiRWjanGxmn96ewBgk9vPTcL6AEZ1t6F6fkgUWe&tokenMint=%s&tokenATA=%s&price=%f&sellerExpiry=-1&useV2=false&buyerCreatorRoyaltyPercent=0`,
				s.privateKey.PublicKey(), token.Seller, token.MintAddress, token.TokenAddress, token.Price)

			signature, err = MintNft(s.privateKey, buyURL)
			if err != nil {
				log.Println("Error while buying nft:", err.Error())
				return
			}
			log.Println("Signature -", signature)
			log.Println("Successfully bought item.")
		}
	}
}

func (s Sniper) parseTransaction(transaction *client2.GetTransactionResponse) *models.Token {
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
