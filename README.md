# LazyQ - Generatore di Domande per Studenti

Un'applicazione desktop per generare automaticamente domande di studio da PDF e immagini utilizzando l'AI di OpenRouter.

## Caratteristiche

- 📄 Estrae testo da file PDF
- 🖼️ Supporta immagini (PNG, JPG, JPEG)  
- 🤖 Genera domande e risposte utilizzando modelli AI avanzati (GPT-4o e altri)
- 💾 Salva domande e risposte in file di testo
- 🎨 Interfaccia grafica intuitiva
- 🔒 Le chiavi API sono salvate localmente in modo sicuro

## Stili di Domande Disponibili

- **Standard**: Domande generali sul contenuto
- **Vero o Falso**: Affermazioni da valutare
- **Sequenziale**: Domande su ordini e sequenze
- **Complicate**: Domande che richiedono analisi approfondita
- **Date e Numeri**: Domande focalizzate su dati numerici

## Requisiti

- Windows (testato su Windows 11)
- Account OpenRouter con crediti (https://openrouter.ai)
- Chiave API di OpenRouter

## Come Ottenere la Chiave API

1. Vai su https://openrouter.ai
2. Crea un account
3. Aggiungi crediti (minimo $5)
4. Vai su "Keys" e crea una nuova chiave API
5. Copia la chiave (inizia con `sk-or-v1-...`)

## Installazione

### Download dell'Eseguibile

1. Vai alla sezione [Releases](../../releases)
2. Scarica `LazyQ.exe` dall'ultima release
3. Esegui il file

### Build dal Codice Sorgente

Requisiti:
- Go 1.21 o superiore
- go-winres (per l'icona)

```bash
# Clona il repository
git clone https://github.com/Exarcun/LazyQ---Genera-Domande-in-Base-Al-Contenuto-Fornito.git
cd LazyQ---Genera-Domande-in-Base-Al-Contenuto-Fornito

# Installa go-winres
go install github.com/tc-hib/go-winres@latest

# Genera i file di risorse Windows (icona)
go-winres make --in winres.json

# Compila usando lo script di build
.\build.bat
```

L'eseguibile sarà creato nella cartella `output/`.

## Utilizzo

1. **Prima Esecuzione**
   - Avvia `LazyQ.exe`
   - Inserisci la tua chiave API di OpenRouter
   - Scegli il modello AI (default: GPT-4o)

2. **Genera Domande**
   - Clicca "Aggiungi PDF/PNG/JPG" per caricare file
   - Scegli il numero di domande (1-100)
   - (Opzionale) Seleziona uno stile di domanda specifico
   - Clicca "Genera Domande"
   - Attendi la generazione (può richiedere alcuni secondi)

3. **Visualizza Risposte**
   - Dopo la generazione, clicca "Mostra Risposte"
   - Le risposte appariranno nella sezione inferiore

4. **Salva Risultati**
   - Clicca "Salva Domande" per esportare
   - Domande e risposte saranno salvate in un file .txt

## Modelli Supportati

Puoi utilizzare qualsiasi modello disponibile su OpenRouter. Alcuni esempi:

- `openai/gpt-4o` (predefinito - ottimo bilanciamento qualità/costo)
- `openai/gpt-4-turbo`
- `anthropic/claude-3.5-sonnet`
- `google/gemini-pro-1.5`
- `meta-llama/llama-3.1-70b-instruct` (più economico)

Consulta https://openrouter.ai/models per l'elenco completo.

## Note Importanti

⚠️ **Le risposte generate dall'AI sono utili ma possono contenere errori o essere incomplete. Si consiglia sempre di consultare il materiale originale per verificare le risposte.**

💰 **Ogni generazione utilizza crediti OpenRouter. Modelli più potenti costano di più ma gestiscono meglio documenti complessi o multipli.**

📄 **Si consiglia di usare file PDF per risultati migliori, poiché l'estrazione del testo è più precisa.**

## Tecnologie Utilizzate

- [Go](https://golang.org/) - Linguaggio di programmazione
- [Fyne](https://fyne.io/) - Framework GUI
- [OpenRouter](https://openrouter.ai/) - API per modelli AI
- [go-winres](https://github.com/tc-hib/go-winres) - Embedding icona Windows

## Struttura del Progetto

```
.
├── main.go              # Codice principale dell'applicazione
├── go.mod               # Dipendenze Go
├── go.sum               # Checksums delle dipendenze
├── winres.json          # Configurazione icona Windows
├── build.bat            # Script di build
├── logoNoBackgorund.ico # Icona dell'applicazione
├── logoNoBackgorund.png # Logo PNG
└── README.md            # Questo file
```

## Licenza

Questo progetto è distribuito "as-is" senza garanzie di alcun tipo.

## Supporto

Per problemi o domande, apri una [Issue](../../issues) su GitHub.

---

Creato con ❤️ per aiutare gli studenti nello studio
