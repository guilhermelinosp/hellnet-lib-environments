# hellnet-lib-environments

> The single, standardized way for Hellnet Go libraries to read configuration
> from environment variables and `.env` files — the shared backend used by
> libs like hellnet-lib-cache and hellnet-lib-telemetry.

## 🧒 Entenda com 15 anos

### A analogia

As env vars são a **lixeira de anotações** presa na porta da geladeira da casa, onde todo mundo cola bilhetes tipo *"senha do wi-fi: XXX"* e *"fone do vizinho: YYY"*. Cada ambiente tem **sua própria lixeira** — seu quarto (*Development*), a casa dos amigos (*Staging*), a escola (*Production*) — com valores diferentes em cada uma, e o programa funciona **igualzinho nos três**.

### O problema que resolve

- Sem uma convenção comum, cada programa inventaria seu próprio jeito de ler configuração — esta lib **é o jeito padrão da família Hellnet** de ler essas anotações (+ arquivo `.env`), sem duplicar código.
- Notas podem faltar ou vir no formato errado: toda leitura tem **fallback/default**, e números, sim/não e tempos são convertidos sempre do mesmo jeito.
- Em desenvolvimento você não quer colar bilhetes na geladeira de verdade: o caderninho **`.env`** vira anotações na porta automaticamente via `LoadDotEnv`.

### Mini-dicionário

| Termo | Analogia |
| --- | --- |
| **env var** | uma anotação pendurada na porta da geladeira. |
| **`.env`** | caderninho local que vira anotações na porta quando você está desenvolvendo. |
| **prefixo `HELLNET_`** | todas as anotações da família começam com `HELLNET_`. |
| **fallback/default** | se a nota não existe, uso o valor combinado. |
| **`GetString`/`GetInt`/`GetBool`/`GetDuration`** | pergunte à lixeira textos, números, sim/não, tempos. |
| **`LoadDotEnv`** | cola o caderninho na porta automaticamente em dev. |

### Primeiras linhas

Quer saber, por exemplo, em qual endereço está o banco?

```go
host := environments.GetString(
	"HELLNET_DATABASE_", // família: a nota começa sempre com HELLNET_ + o nome da lib
	"",                  // sub-família: vazio = nenhuma (poderia ser "CACHE_" ou "DB_")
	"HOST",              // nome da nota: junto com os prefixos acima, lê HELLNET_DATABASE_HOST
	"localhost",         // fallback: valor combinado caso não encontre essa nota na lixeira
)
```

## Uso

🧒 *Na analogia: você **pergunta à lixeira** — "existe uma nota chamada X?" — e, se ninguém escreveu nada, usa o valor combinado.*

```go
import (
	"time"

	"github.com/guilhermelinosp/hellnet-lib-environments/environments"
)

func main() {
	// Em desenvolvimento, carrega o primeiro .env confiável encontrado
	// (ao lado do executável ou na pasta atual e subindo pelos pais).
	_ = environments.LoadDotEnv()

	host := environments.GetString("HELLNET_CACHE_", "", "HOST", "localhost")
	port := environments.GetInt("HELLNET_CACHE_", "HELLNET_DATABASE_", "PORT", 5432)
	debug := environments.GetBool("HELLNET_CACHE_", "", "DEBUG", false)
	ttl := environments.GetDuration("HELLNET_CACHE_", "", "TTL", time.Minute)
}
```

Cada getter recebe `(prefix, fallbackPrefix, suffix, def)` e devolve o primeiro valor **não vazio** entre `prefix+suffix`, `fallbackPrefix+suffix` e `def`. Por isso a convenção de nomes é **`HELLNET_<LIB>_`**: a lib `_CACHE_` lê as notas dela e pode usar outra família como fallback. As variantes `GetIntE`/`GetBoolE`/`GetDurationE` também retornam um erro quando a nota existe mas está num formato inválido.

## Develop

```bash
make all        # fmt + vet + lint + test
make test-race  # tests with the race detector
make cover      # coverage report (coverage.out)
```

Install the git hooks once:

```bash
lefthook install
```

## Conventional Commits

Commits and PR titles follow [Conventional Commits](https://www.conventionalcommits.org/).
Releases and version bumps are derived from them — see `CONTRIBUTING.md`.

## License

[Apache 2.0](LICENSE)
