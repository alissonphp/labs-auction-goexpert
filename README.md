# Leilão com fechamento automático

Projeto base do desafio (`labs-auction-goexpert`) com a rotina de fechamento automático dos leilões implementada.

Assim que um leilão é criado, uma goroutine é disparada em `internal/infra/database/auction/create_auction.go`. Ela aguarda o tempo configurado em `AUCTION_DURATION` e, quando esse tempo expira, atualiza o status do leilão para `Completed` diretamente no MongoDB — sem bloquear a requisição de criação.

## Como rodar

```bash
docker compose up -d --build
```

Sobem dois containers: a aplicação na porta `8080` e o MongoDB na `27017`.

Criando um leilão:

```bash
curl -X POST http://localhost:8080/auction \
  -H "Content-Type: application/json" \
  -d '{
    "product_name": "Notebook",
    "category": "eletronicos",
    "description": "notebook usado em bom estado",
    "condition": 1
  }'
```

Listando os leilões abertos (`status=0`) e fechados (`status=1`):

```bash
curl "http://localhost:8080/auction?status=0"
curl "http://localhost:8080/auction?status=1"
```

Crie um leilão, espere o tempo de `AUCTION_DURATION` e consulte novamente: ele terá saído da lista de abertos e aparecido na de fechados. O log da aplicação registra `Auction <id> closed automatically`.

## Variáveis de ambiente

Ficam em `cmd/auction/.env`:

| Variável | Padrão | Descrição |
| --- | --- | --- |
| `AUCTION_DURATION` | `20s` | Duração do leilão. É o tempo que a goroutine espera antes de fechá-lo. Se estiver ausente ou inválida, o código usa 5 minutos |
| `AUCTION_INTERVAL` | `20s` | Usado pela validação de lances para calcular o fim do leilão. Mantenha com o mesmo valor de `AUCTION_DURATION`, senão lances continuam sendo aceitos depois do fechamento |
| `BATCH_INSERT_INTERVAL` | `20s` | Intervalo de gravação dos lances em lote |
| `MAX_BATCH_SIZE` | `4` | Tamanho do lote de lances |
| `MONGODB_URL` | `mongodb://admin:admin@mongodb:27017/auctions?authSource=admin` | Conexão com o MongoDB |
| `MONGODB_DB` | `auctions` | Banco utilizado |

O valor aceita qualquer duração no formato do `time.ParseDuration`: `30s`, `5m`, `1h`.

## Versão do MongoDB

O compose fixa `mongo:7.0`. O MongoDB 8.x não sobe em kernels Linux 6.19+ por causa de um conflito entre o TCMalloc que ele embarca e a ABI de `rseq` do kernel ([SERVER-121912](https://jira.mongodb.org/browse/SERVER-121912)), e o Docker Desktop recente já usa um kernel dessa faixa. A imagem 7.0 é anterior ao TCMalloc afetado e funciona normalmente.

Para usar outra versão, sobrescreva a variável:

```bash
MONGO_IMAGE_TAG=8.0 docker compose up -d
```

## Testes

O teste de fechamento automático é de integração: ele cria um leilão de verdade, espera a expiração e confere o status no banco. Por isso o MongoDB precisa estar no ar.

```bash
docker compose up -d mongodb
MONGODB_URL="mongodb://admin:admin@localhost:27017/auctions?authSource=admin" go test ./internal/infra/database/auction/ -v
```

O teste define `AUCTION_DURATION=3s` por conta própria, então não depende do valor do `.env`. Se o MongoDB não estiver acessível, ele é marcado como `SKIP` em vez de falhar.

O que o teste valida:

1. cria o leilão e confere que ele nasceu com status `Active`;
2. aguarda a expiração;
3. busca o leilão de novo e exige status `Completed`, sem nenhuma intervenção manual.
