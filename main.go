package main

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	pdf "github.com/ledongthuc/pdf"
)

//go:embed logoNoBackgorund.png
var logoData []byte

const (
	appID         = "lazyq"
	appTitle      = "LazyQ"
	defaultModel  = "openai/gpt-4o" // Puoi cambiare in "openai/gpt-5" se disponibile sul tuo account OpenRouter
	maxTextChars  = 12000           // sicurezza per mantenere le dimensioni del payload ragionevoli
	openRouterURL = "https://openrouter.ai/api/v1/chat/completions"
	prefAPIKey    = "openrouter_api_key"
	prefModel     = "openrouter_model"
	defaultN      = 10
	refererHeader = "https://local-app/lazyq"
	xTitleHeader  = "LazyQ"
)

func main() {
	a := app.NewWithID(appID)

	// Set app icon from embedded data
	iconRes := fyne.NewStaticResource("logo.png", logoData)
	a.SetIcon(iconRes)

	w := a.NewWindow(appTitle)
	w.Resize(fyne.NewSize(900, 650))

	// Set window icon
	w.SetIcon(iconRes)

	// Navigate through simple "screens"
	var setContent func(fyne.CanvasObject)
	setContent = func(co fyne.CanvasObject) {
		w.SetContent(container.NewMax(co))
	}

	// Check if API key exists
	prefs := a.Preferences()
	existingKey := strings.TrimSpace(prefs.String(prefAPIKey))

	if existingKey != "" {
		// Skip greet and key screen, go directly to main
		setContent(createMainScreen(a, w, func() {
			setContent(createAPIKeyScreen(a, w, func() {
				setContent(createMainScreen(a, w, nil))
			}))
		}))
	} else {
		// Show greet screen first
		greet := createGreetScreen(w, func() {
			setContent(createAPIKeyScreen(a, w, func() {
				setContent(createMainScreen(a, w, nil))
			}))
		})
		setContent(greet)
	}

	w.ShowAndRun()
}

func createGreetScreen(w fyne.Window, onNext func()) fyne.CanvasObject {
	title := widget.NewLabelWithStyle("Benvenuto al Generatore di Test per Studenti", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	sub := widget.NewLabel("Genera domande di studio dai tuoi PDF e immagini. Continua per inserire la tua chiave API di OpenRouter.")

	btn := widget.NewButtonWithIcon("Continua", theme.NavigateNextIcon(), func() {
		onNext()
	})
	return container.NewBorder(
		container.NewVBox(
			container.NewCenter(title),
			container.NewCenter(sub),
		),
		container.NewCenter(btn),
		nil, nil,
	)
}

func createAPIKeyScreen(a fyne.App, w fyne.Window, onNext func()) fyne.CanvasObject {
	prefs := a.Preferences()
	existing := prefs.String(prefAPIKey)
	modelExisting := prefs.String(prefModel)
	if modelExisting == "" {
		modelExisting = defaultModel
	}

	info := widget.NewLabel("Inserisci la tua chiave API di OpenRouter. Sarà salvata localmente nelle preferenze dell'app.")
	entry := widget.NewPasswordEntry()
	entry.SetPlaceHolder("sk-or-v1-...")
	entry.SetText(existing)

	modelLabel := widget.NewLabel("Modello (ID OpenRouter):")
	modelEntry := widget.NewEntry()
	modelEntry.SetPlaceHolder(defaultModel)
	modelEntry.SetText(modelExisting)

	// Helper guide button
	helpBtn := widget.NewButtonWithIcon("Come ottenere la chiave API?", theme.HelpIcon(), func() {
		helpText := `GUIDA: Come Ottenere la Chiave API di OpenRouter

1. VAI SU OPENROUTER
   • Apri il browser e vai su: https://openrouter.ai

2. CREA UN ACCOUNT
   • Clicca su "Sign In" in alto a destra
   • Scegli tra Google, GitHub o Email
   • Completa la registrazione

3. AGGIUNGI CREDITI
   • Una volta loggato, vai su "Credits" nel menu
   • Clicca su "Add Credits"
   • Scegli l'importo (minimo $5)
   • Completa il pagamento
   • I crediti vengono usati per pagare le richieste AI

4. CREA UNA CHIAVE API
   • Vai su "Keys" nel menu (o "API Keys")
   • Clicca su "Create Key" o "+ New Key"
   • Dai un nome alla chiave (es. "Test Generator")
   • Opzionale: imposta limiti di spesa
   • Clicca su "Create"
   • La chiave inizia con "sk-or-v1-..."

5. COPIA E INCOLLA
   • Copia la chiave API (mostrata una sola volta!)
   • Incollala nel campo qui sotto
   • Clicca su "Salva e Continua"

NOTA: La chiave API è come una password. Non condividerla!
Il modello predefinito è GPT-4o, ma puoi cambiarlo.`

		dialog.ShowInformation("Guida OpenRouter", helpText, w)
	})

	save := widget.NewButtonWithIcon("Salva e Continua", theme.ConfirmIcon(), func() {
		key := strings.TrimSpace(entry.Text)
		if key == "" {
			dialog.ShowInformation("Chiave Mancante", "Inserisci una chiave API di OpenRouter valida.", w)
			return
		}
		prefs.SetString(prefAPIKey, key)

		model := strings.TrimSpace(modelEntry.Text)
		if model == "" {
			model = defaultModel
		}
		prefs.SetString(prefModel, model)

		onNext()
	})

	return container.NewVBox(
		widget.NewLabelWithStyle("Configurazione API", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		info,
		entry,
		modelLabel,
		modelEntry,
		helpBtn,
		save,
	)
}

func createMainScreen(a fyne.App, w fyne.Window, onSettings func()) fyne.CanvasObject {
	prefs := a.Preferences()
	model := strings.TrimSpace(prefs.String(prefModel))
	if model == "" {
		model = defaultModel
	}

	// State
	var selectedNames []string
	var selectedTexts []string
	var selectedImages []string // data URLs for images

	namesLabel := widget.NewLabel("Nessun file selezionato.")
	updateNames := func() {
		if len(selectedNames) == 0 {
			namesLabel.SetText("Nessun file selezionato.")
		} else {
			namesLabel.SetText(strings.Join(selectedNames, "\n"))
		}
	}

	// Controls
	nEntry := widget.NewEntry()
	nEntry.SetPlaceHolder(fmt.Sprintf("%d", defaultN))
	nEntry.SetText(fmt.Sprintf("%d", defaultN))

	modelEntry := widget.NewEntry()
	modelEntry.SetPlaceHolder(defaultModel)
	modelEntry.SetText(model)

	addFileBtn := widget.NewButtonWithIcon("Aggiungi PDF/PNG/JPG", theme.FileIcon(), func() {
		fd := dialog.NewFileOpen(func(r fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if r == nil {
				return
			}
			defer r.Close()

			uri := r.URI()
			name := filepath.Base(uri.Path())
			ext := strings.ToLower(filepath.Ext(name))

			// Read content to memory
			data, rdErr := io.ReadAll(r)
			if rdErr != nil {
				dialog.ShowError(fmt.Errorf("failed reading file: %w", rdErr), w)
				return
			}

			switch ext {
			case ".pdf":
				text, perr := extractTextFromPDFBytes(data)
				if perr != nil {
					dialog.ShowError(fmt.Errorf("PDF parse error: %w", perr), w)
					return
				}
				if strings.TrimSpace(text) == "" {
					dialog.ShowInformation("PDF Vuoto", "Nessun testo estraibile trovato in questo PDF.", w)
					return
				}
				selectedTexts = append(selectedTexts, text)
				selectedNames = append(selectedNames, fmt.Sprintf("PDF: %s (%.1f KB)", name, float64(len(data))/1024))
				updateNames()

			case ".png", ".jpg", ".jpeg":
				dataURL, ierr := imageBytesToDataURL(data, ext)
				if ierr != nil {
					dialog.ShowError(ierr, w)
					return
				}
				selectedImages = append(selectedImages, dataURL)
				selectedNames = append(selectedNames, fmt.Sprintf("Image: %s (%.1f KB)", name, float64(len(data))/1024))
				updateNames()

			default:
				dialog.ShowInformation("Non Supportato", "Scegli un file PDF, PNG, JPG o JPEG.", w)
			}
		}, w)
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".pdf", ".png", ".jpg", ".jpeg"}))
		fd.Show()
	})

	questionsOutput := widget.NewMultiLineEntry()
	questionsOutput.SetPlaceHolder("Le domande generate appariranno qui...")
	questionsOutput.Wrapping = fyne.TextWrapWord
	questionsOutput.Disable()

	answersOutput := widget.NewMultiLineEntry()
	answersOutput.SetPlaceHolder("Le risposte appariranno qui dopo aver cliccato 'Mostra Risposte'...")
	answersOutput.Wrapping = fyne.TextWrapWord
	answersOutput.Disable()

	var currentAnswers string
	var answersVisible bool

	showAnswersBtn := widget.NewButtonWithIcon("Mostra Risposte", theme.VisibilityIcon(), nil)
	showAnswersBtn.OnTapped = func() {
		if currentAnswers == "" {
			dialog.ShowInformation("Nessuna Risposta", "Genera prima le domande per vedere le risposte.", w)
			return
		}
		if answersVisible {
			// Hide answers
			answersOutput.SetText("")
			answersOutput.Disable()
			showAnswersBtn.SetText("Mostra Risposte")
			showAnswersBtn.SetIcon(theme.VisibilityIcon())
			answersVisible = false
		} else {
			// Show answers
			answersOutput.SetText(currentAnswers)
			answersOutput.Enable()
			showAnswersBtn.SetText("Nascondi Risposte")
			showAnswersBtn.SetIcon(theme.VisibilityOffIcon())
			answersVisible = true
		}
		answersOutput.Refresh()
		showAnswersBtn.Refresh()
	}
	showAnswersBtn.Disable()

	clearBtn := widget.NewButtonWithIcon("Cancella", theme.ContentClearIcon(), nil)
	clearBtn.Importance = widget.DangerImportance
	clearBtn.OnTapped = func() {
		selectedNames = nil
		selectedTexts = nil
		selectedImages = nil
		updateNames()
		questionsOutput.SetText("")
		answersOutput.SetText("")
		currentAnswers = ""
		answersVisible = false
		showAnswersBtn.SetText("Mostra Risposte")
		showAnswersBtn.SetIcon(theme.VisibilityIcon())
		showAnswersBtn.Disable()
	}

	// Question styles section (declared early for genBtn to use)
	styleCheckbox := widget.NewCheck("Stili risposte", nil)

	styleRadio := widget.NewRadioGroup([]string{
		"Vero o Falso",
		"Sequenziale",
		"Complicate",
		"Date e numeri",
	}, nil)
	styleRadio.Disable()

	styleCheckbox.OnChanged = func(checked bool) {
		if checked {
			styleRadio.Enable()
			if styleRadio.Selected == "" {
				styleRadio.SetSelected("Vero o Falso")
			}
		} else {
			styleRadio.Disable()
		}
	}

	saveBtn := widget.NewButtonWithIcon("Salva Domande", theme.DocumentSaveIcon(), func() {
		if strings.TrimSpace(questionsOutput.Text) == "" {
			dialog.ShowInformation("Niente da Salvare", "Esegui prima la generazione per produrre domande.", w)
			return
		}
		fs := dialog.NewFileSave(func(wc fyne.URIWriteCloser, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if wc == nil {
				return
			}
			defer wc.Close()
			content := questionsOutput.Text
			if currentAnswers != "" {
				content += "\n\n=== RISPOSTE ===\n\n" + currentAnswers
			}
			if _, err := wc.Write([]byte(content)); err != nil {
				dialog.ShowError(err, w)
				return
			}
			dialog.ShowInformation("Salvato", "Output salvato con successo.", w)
		}, w)
		fs.SetFileName("domande_risposte.txt")
		fs.Show()
	})

	genBtn := widget.NewButtonWithIcon("Genera Domande", theme.MediaPlayIcon(), nil)
	genBtn.OnTapped = func() {
		// Validation
		key := strings.TrimSpace(prefs.String(prefAPIKey))
		if key == "" {
			dialog.ShowInformation("Chiave API Mancante", "Imposta la tua chiave API di OpenRouter nel passaggio precedente.", w)
			return
		}
		modelStr := strings.TrimSpace(modelEntry.Text)
		if modelStr == "" {
			modelStr = defaultModel
		}
		nStr := strings.TrimSpace(nEntry.Text)
		if nStr == "" {
			nStr = fmt.Sprintf("%d", defaultN)
		}
		nVal, err := strconv.Atoi(nStr)
		if err != nil || nVal < 1 || nVal > 100 {
			dialog.ShowInformation("Numero Non Valido", "Inserisci un numero valido di domande (1-100).", w)
			return
		}
		if len(selectedTexts) == 0 && len(selectedImages) == 0 {
			dialog.ShowInformation("Nessuna Fonte", "Aggiungi almeno un PDF o un'immagine.", w)
			return
		}

		// Persist model choice
		prefs.SetString(prefModel, modelStr)

		// Prepare UI
		genBtn.Disable()
		showAnswersBtn.Disable()
		questionsOutput.SetText("Generazione in corso... Potrebbe richiedere un momento.")
		questionsOutput.Refresh()
		answersOutput.SetText("")
		answersOutput.Refresh()

		go func() {
			start := time.Now()
			questionStyle := ""
			if styleCheckbox.Checked {
				questionStyle = styleRadio.Selected
			}
			questions, answers, cost, gErr := generateQuestionsAndAnswers(key, modelStr, nVal, selectedTexts, selectedImages, questionStyle)
			elapsed := time.Since(start)

			// Update UI
			genBtn.Enable()
			if gErr != nil {
				questionsOutput.SetText(fmt.Sprintf("Errore: %v", gErr))
				currentAnswers = ""
			} else {
				questionsOutput.SetText(fmt.Sprintf("%s\n\n--\nGenerato in %s", strings.TrimSpace(questions), elapsed.Truncate(time.Millisecond)))
				currentAnswers = answers
				showAnswersBtn.Enable()
				_ = cost // ignore cost for now
			}
			questionsOutput.Enable() // allow copy
			questionsOutput.Refresh()
		}()
	}

	// Settings button (only show if callback provided)
	var settingsBtn *widget.Button
	if onSettings != nil {
		settingsBtn = widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
			onSettings()
		})
	}

	header := widget.NewLabelWithStyle("Principale", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	header2 := widget.NewLabel("Aggiungi PDF e/o immagini, scegli il modello e il numero di domande, poi Genera.")

	controls := container.NewHBox(addFileBtn, clearBtn)
	params := container.NewGridWithColumns(2,
		widget.NewLabel("Modello:"), modelEntry,
		widget.NewLabel("Numero di Domande:"), nEntry,
	)

	// Expanded file list with min height
	fileListScroll := container.NewVScroll(namesLabel)
	fileListScroll.SetMinSize(fyne.NewSize(0, 150))

	// Helper texts
	helpText1 := widget.NewLabel("Le risposte generate non seguono completamente il materiale fornito, sono un aiuto solido ma possono contenere errori e risposte parziali. Si consiglia di consultare il materiale fornito per le domande per un auto controllo efficiente.")
	helpText1.Wrapping = fyne.TextWrapWord

	helpText2 := widget.NewLabel("Ogni generazione utilizza il credito di OpenRouter. Puoi anche utilizzare modelli meno precisi per un costo più basso, oppure modelli più costosi ma che riescono a gestire un numero maggiore di documenti.")
	helpText2.Wrapping = fyne.TextWrapWord

	pdfHint := widget.NewLabel("Si consiglia di usare PDF")
	pdfHint.TextStyle = fyne.TextStyle{Italic: true}

	var headerContent fyne.CanvasObject
	if settingsBtn != nil {
		headerContent = container.NewBorder(nil, nil, nil, settingsBtn, container.NewVBox(header, header2))
	} else {
		headerContent = container.NewVBox(header, header2)
	}

	left := container.NewVBox(
		headerContent,
		widget.NewSeparator(),
		container.NewHBox(
			widget.NewLabelWithStyle("File Selezionati:", fyne.TextAlignLeading, fyne.TextStyle{Bold: false}),
			pdfHint,
		),
		fileListScroll,
		controls,
		params,
		widget.NewSeparator(),
		styleCheckbox,
		styleRadio,
		widget.NewSeparator(),
		genBtn,
		saveBtn,
		widget.NewSeparator(),
		helpText1,
		helpText2,
	)

	// Vertical split for questions and answers with "Mostra Risposte" button in answer section
	rightPanel := container.NewVSplit(
		container.NewBorder(
			widget.NewLabel("Domande:"),
			nil, nil, nil,
			container.NewMax(container.NewVScroll(questionsOutput)),
		),
		container.NewBorder(
			container.NewHBox(widget.NewLabel("Risposte:"), showAnswersBtn),
			nil, nil, nil,
			container.NewMax(container.NewVScroll(answersOutput)),
		),
	)
	rightPanel.Offset = 0.5

	content := container.NewHSplit(
		container.NewVScroll(left),
		rightPanel,
	)
	content.Offset = 0.35

	return content
}

func extractTextFromPDFBytes(data []byte) (string, error) {
	rd := bytes.NewReader(data)
	pdfReader, err := pdf.NewReader(rd, int64(len(data)))
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	pt, err := pdfReader.GetPlainText()
	if err != nil {
		return "", err
	}
	_, _ = io.Copy(&buf, pt)
	text := buf.String()
	// trim and clamp size
	text = strings.TrimSpace(text)
	if len(text) > maxTextChars {
		text = text[:maxTextChars] + "\n...[truncated]..."
	}
	return text, nil
}

func imageBytesToDataURL(data []byte, ext string) (string, error) {
	// Normalize ext to mime
	ext = strings.ToLower(ext)
	mtyp := mime.TypeByExtension(ext)
	if mtyp == "" {
		switch ext {
		case ".jpg", ".jpeg":
			mtyp = "image/jpeg"
		case ".png":
			mtyp = "image/png"
		default:
			mtyp = "application/octet-stream"
		}
	}
	enc := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", mtyp, enc), nil
}

// OpenRouter types

type contentPart struct {
	Type     string    `json:"type"`                // "text" or "image_url"
	Text     string    `json:"text,omitempty"`      // for text
	ImageURL *imageURL `json:"image_url,omitempty"` // for image
}

type imageURL struct {
	URL string `json:"url"`
}

type message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // string or []contentPart
}

type chatRequest struct {
	Model       string    `json:"model"`
	Messages    []message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
}

type chatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

func generateQuestions(apiKey, model string, n int, texts []string, imageDataURLs []string) (string, error) {
	systemPrompt := "Sei un insegnante esperto. Dal materiale di studio fornito, estrai prima i punti principali, poi formula domande numerate che riflettono quei punti. Usa solo il contenuto fornito; non aggiungere contesto esterno. Rispondi SEMPRE in italiano."

	// Build text body
	var mergedText string
	if len(texts) > 0 {
		// Join with delimiter
		mergedText = strings.Join(texts, "\n\n---\n\n")
		if len(mergedText) > maxTextChars {
			mergedText = mergedText[:maxTextChars] + "\n...[troncato]..."
		}
	}

	// Build user content array
	var parts []contentPart
	var b strings.Builder

	// Instruction to ensure strict format
	fmt.Fprintf(&b, "Istruzioni:\n- Estrai i punti principali dal materiale fornito.\n- Poi produci esattamente %d domande IN ITALIANO.\n- Le domande devono usare questo formato rigoroso:\n", n)
	fmt.Fprintf(&b, "1. Prima domanda\n2. Seconda domanda\n3. Terza domanda\n...\n")
	b.WriteString("- Non includere risposte.\n- Non includere testo introduttivo o conclusivo.\n- Non aggiungere conoscenze esterne.\n- Fai domande solo sulle informazioni presenti nel materiale fornito.\n- Scrivi TUTTO in italiano.\n\n")

	if strings.TrimSpace(mergedText) != "" {
		b.WriteString("Materiale di studio (testo):\n")
		b.WriteString(mergedText)
	}
	// Add as one text part
	parts = append(parts, contentPart{Type: "text", Text: b.String()})

	// Add images
	for _, du := range imageDataURLs {
		parts = append(parts, contentPart{
			Type:     "image_url",
			ImageURL: &imageURL{URL: du},
		})
	}

	reqBody := chatRequest{
		Model: model,
		Messages: []message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: parts},
		},
		Temperature: 0.2,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequest(http.MethodPost, openRouterURL, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("HTTP-Referer", refererHeader)
	httpReq.Header.Set("X-Title", xTitleHeader)

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("OpenRouter HTTP %d: %s", resp.StatusCode, truncate(string(body), 500))
	}

	var cr chatResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		// Try to surface raw body if unmarshalling fails
		return "", fmt.Errorf("failed to decode response: %v\nRaw: %s", err, truncate(string(body), 800))
	}
	if cr.Error != nil {
		return "", fmt.Errorf("OpenRouter error: %s (%s)", cr.Error.Message, cr.Error.Type)
	}
	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("no choices returned")
	}
	return cr.Choices[0].Message.Content, nil
}

func generateQuestionsAndAnswers(apiKey, model string, n int, texts []string, imageDataURLs []string, questionStyle string) (string, string, float64, error) {
	// Build merged text
	var mergedText string
	if len(texts) > 0 {
		mergedText = strings.Join(texts, "\n\n---\n\n")
		if len(mergedText) > maxTextChars {
			mergedText = mergedText[:maxTextChars] + "\n...[troncato]..."
		}
	}

	// Build content parts
	var parts []contentPart
	var b strings.Builder

	// Different prompts based on question style
	switch questionStyle {
	case "Vero o Falso":
		fmt.Fprintf(&b, "Istruzioni:\n- Estrai i punti principali dal materiale fornito.\n- Produci esattamente %d domande VERO o FALSO IN ITALIANO.\n- Usa questo formato RIGOROSO:\n\n", n)
		b.WriteString("DOMANDE:\n1. [Affermazione che può essere vera o falsa]\n2. [Affermazione che può essere vera o falsa]\n...\n\n")
		b.WriteString("RISPOSTE:\n1. Vero / Falso (con breve spiegazione)\n2. Vero / Falso (con breve spiegazione)\n...\n\n")
	case "Sequenziale":
		fmt.Fprintf(&b, "Istruzioni:\n- Estrai eventi, processi o passaggi sequenziali dal materiale fornito.\n- Produci esattamente %d domande SEQUENZIALI IN ITALIANO che richiedono di ordinare o descrivere una sequenza.\n- Usa questo formato RIGOROSO:\n\n", n)
		b.WriteString("DOMANDE:\n1. Qual è la sequenza corretta di...?\n2. Metti in ordine i seguenti passaggi...\n...\n\n")
		b.WriteString("RISPOSTE:\n1. La sequenza corretta è: ...\n2. L'ordine corretto è: ...\n...\n\n")
	case "Complicate":
		fmt.Fprintf(&b, "Istruzioni:\n- Estrai concetti complessi e relazioni dal materiale fornito.\n- Produci esattamente %d domande COMPLESSE IN ITALIANO che richiedono analisi approfondita, confronto, o sintesi di più concetti.\n- Usa questo formato RIGOROSO:\n\n", n)
		b.WriteString("DOMANDE:\n1. Spiega la relazione tra... e come...\n2. Confronta e analizza...\n3. Perché... e quali sono le implicazioni di...\n...\n\n")
		b.WriteString("RISPOSTE:\n1. [Risposta articolata e dettagliata]\n2. [Risposta articolata e dettagliata]\n...\n\n")
	case "Date e numeri":
		fmt.Fprintf(&b, "Istruzioni:\n- Estrai date, numeri, statistiche e dati numerici specifici dal materiale fornito.\n- Produci esattamente %d domande IN ITALIANO incentrate su DATE e NUMERI.\n- Usa questo formato RIGOROSO:\n\n", n)
		b.WriteString("DOMANDE:\n1. In che anno...?\n2. Quanti...?\n3. Qual è la percentuale di...?\n...\n\n")
		b.WriteString("RISPOSTE:\n1. [Anno o data specifica]\n2. [Numero specifico]\n3. [Percentuale o valore numerico]\n...\n\n")
	default:
		// Standard format
		fmt.Fprintf(&b, "Istruzioni:\n- Estrai i punti principali dal materiale fornito.\n- Produci esattamente %d domande IN ITALIANO con le relative risposte.\n- Usa questo formato RIGOROSO:\n\n", n)
		b.WriteString("DOMANDE:\n1. Prima domanda\n2. Seconda domanda\n3. Terza domanda\n...\n\n")
		b.WriteString("RISPOSTE:\n1. Risposta alla prima domanda\n2. Risposta alla seconda domanda\n3. Risposta alla terza domanda\n...\n\n")
	}

	b.WriteString("- Non aggiungere testo introduttivo o conclusivo.\n- Scrivi TUTTO in italiano.\n- Usa SOLO informazioni dal materiale fornito.\n\n")

	if strings.TrimSpace(mergedText) != "" {
		b.WriteString("Materiale di studio:\n")
		b.WriteString(mergedText)
	}

	parts = append(parts, contentPart{Type: "text", Text: b.String()})

	for _, du := range imageDataURLs {
		parts = append(parts, contentPart{
			Type:     "image_url",
			ImageURL: &imageURL{URL: du},
		})
	}

	reqBody := chatRequest{
		Model: model,
		Messages: []message{
			{Role: "system", Content: "Sei un insegnante esperto. Genera domande e risposte in italiano dal materiale fornito."},
			{Role: "user", Content: parts},
		},
		Temperature: 0.2,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", "", 0, err
	}

	httpReq, err := http.NewRequest(http.MethodPost, openRouterURL, bytes.NewReader(payload))
	if err != nil {
		return "", "", 0, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("HTTP-Referer", refererHeader)
	httpReq.Header.Set("X-Title", xTitleHeader)

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", "", 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", 0, fmt.Errorf("OpenRouter HTTP %d: %s", resp.StatusCode, truncate(string(body), 500))
	}

	var cr chatResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		return "", "", 0, fmt.Errorf("failed to decode response: %v\nRaw: %s", err, truncate(string(body), 800))
	}
	if cr.Error != nil {
		return "", "", 0, fmt.Errorf("OpenRouter error: %s (%s)", cr.Error.Message, cr.Error.Type)
	}
	if len(cr.Choices) == 0 {
		return "", "", 0, fmt.Errorf("no choices returned")
	}

	fullResponse := cr.Choices[0].Message.Content

	// Parse questions and answers
	questions := ""
	answers := ""

	parts2 := strings.Split(fullResponse, "RISPOSTE:")
	if len(parts2) >= 2 {
		questionPart := strings.TrimSpace(parts2[0])
		answerPart := strings.TrimSpace(parts2[1])

		questionPart = strings.TrimPrefix(questionPart, "DOMANDE:")
		questions = strings.TrimSpace(questionPart)
		answers = answerPart
	} else {
		questions = fullResponse
		answers = "Risposte non disponibili"
	}

	// Estimate cost (approximate based on typical pricing)
	// OpenRouter charges vary, but $0.01 per 1K tokens is a rough estimate for GPT-4o
	estimatedCost := 0.0 // We can't get exact cost from response, so leaving at 0

	return questions, answers, estimatedCost, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "...[truncated]"
}

// Optional: allow running headless to test API without UI
func init() {
	if len(os.Args) > 1 && os.Args[1] == "--selftest" {
		key := os.Getenv("OPENROUTER_API_KEY")
		if key == "" {
			fmt.Println("Set OPENROUTER_API_KEY to run --selftest")
			os.Exit(1)
		}
		resp, err := generateQuestions(key, defaultModel, 5, []string{"Photosynthesis converts light energy to chemical energy in plants. Chlorophyll, thylakoid membranes, and the Calvin cycle are key components."}, nil)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(2)
		}
		fmt.Println(resp)
		os.Exit(0)
	}
}
