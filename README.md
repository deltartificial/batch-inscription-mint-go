# batch-inscription-mint-go
 📜 Batch Mint Inscription over EVM networks in Go

## Setup

Before running the application, ensure you have Go installed on your machine. Then, follow these steps to configure your environment.

### 1. Clone the Repository

Clone the repository to your local machine:

```bash
git clone https://github.com/deltartificial/batch-inscription-mint-go
cd batch-inscription-mint-go
```

### 2. Install Dependencies

Install the required Go dependencies:

```bash
go mod tidy
```

or

```bash
go get .
```

### 3. Configure `.env` File

Create a `.env` file in the root of the project directory with the following variables:

or, `export HTTP_RPC_URL=http://localhost:8545`, same for each one.

```plaintext
HTTP_RPC_URL=http://localhost:8545
PRIVATE_KEY_HEX=your_private_key_hex_here
NUM_WORKERS=5
TRANSACTIONS_NUMBER=50
JSON_DATA="data:,{\"p\":\"asc-20\",\"op\":\"mint\",\"tick\":\"BTC\",\"amt\":\"1\"}"
```

- `HTTP_RPC_URL`: The URL of your Ethereum client's HTTP RPC endpoint.
- `PRIVATE_KEY_HEX`: The private key hex string of the Ethereum account sending the transactions.
- `NUM_WORKERS`: The number of worker goroutines to use for sending transactions.
- `TRANSACTIONS_NUMBER`: The total number of transactions to send.
- `JSON_DATA`: The JSON data to include with each transaction.

### 4. Run the Application

Ensure the `.env` file is configured correctly, then run the application:

```bash
go run main.go
```

@author - deltartificial