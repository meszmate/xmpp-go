package register

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// NS is the namespace for XEP-0077 In-Band Registration
const NS = "jabber:iq:register"

// RegistrationField represents a field in the registration form
type RegistrationField struct {
	Name     string // username, password, email, etc.
	Label    string // Human-readable label
	Required bool
	Password bool   // Mask input
	Type     string // text-single, text-private, hidden, fixed, etc.
	Value    string // Pre-filled value (for hidden fields)
}

// CaptchaData holds CAPTCHA information
type CaptchaData struct {
	Type      string   // "image", "audio", "video", "qa", "hashcash"
	Challenge string   // Challenge type from XEP-0158 (ocr, audio_recog, picture_q, etc.)
	MimeType  string   // e.g., "image/png", "audio/wav"
	Data      []byte   // Base64-decoded media data
	URLs      []string // URLs to fetch CAPTCHA from (may have multiple alternatives)
	URL       string   // Primary URL (first http(s) URL or empty)
	Question  string   // For text-based QA CAPTCHAs or challenge description
	FieldVar  string   // The field var name for submitting the answer
}

// RegistrationForm represents the registration form from the server
type RegistrationForm struct {
	Server          string
	Port            int
	Instructions    string
	Fields          []RegistrationField
	IsDataForm      bool   // True if using XEP-0004 Data Forms
	FormType        string // FORM_TYPE value if present
	RequiresCaptcha bool   // True if CAPTCHA is required
	Captcha         *CaptchaData
}

// RegistrationResult represents the result of a registration attempt
type RegistrationResult struct {
	Success bool
	JID     string
	Error   string
}

// Common field names used in XEP-0077
var fieldLabels = map[string]string{
	"username":   "Username",
	"password":   "Password",
	"email":      "Email",
	"name":       "Full Name",
	"first":      "First Name",
	"last":       "Last Name",
	"nick":       "Nickname",
	"address":    "Address",
	"city":       "City",
	"state":      "State",
	"zip":        "ZIP Code",
	"phone":      "Phone",
	"url":        "Website",
	"date":       "Date",
	"misc":       "Miscellaneous",
	"text":       "Text",
	"key":        "Key",
	"registered": "Already Registered",
}

// passwordFields are fields that should be masked
var passwordFields = map[string]bool{
	"password": true,
}

// streamFeatures represents stream features from server
type streamFeatures struct {
	XMLName    xml.Name `xml:"features"`
	StartTLS   *startTLS
	Mechanisms *mechanisms
	Register   *registerFeature
}

type startTLS struct {
	XMLName  xml.Name  `xml:"starttls"`
	Required *struct{} `xml:"required"`
}

type mechanisms struct {
	XMLName   xml.Name `xml:"mechanisms"`
	Mechanism []string `xml:"mechanism"`
}

type registerFeature struct {
	XMLName xml.Name `xml:"register"`
}

// iqStanza represents an IQ stanza
type iqStanza struct {
	XMLName xml.Name       `xml:"iq"`
	Type    string         `xml:"type,attr"`
	ID      string         `xml:"id,attr"`
	To      string         `xml:"to,attr,omitempty"`
	From    string         `xml:"from,attr,omitempty"`
	Query   *registerQuery `xml:"query,omitempty"`
	Error   *stanzaError   `xml:"error,omitempty"`
}

type registerQuery struct {
	XMLName      xml.Name   `xml:"query"`
	XMLNS        string     `xml:"xmlns,attr"`
	Instructions string     `xml:"instructions,omitempty"`
	Username     *string    `xml:"username,omitempty"`
	Password     *string    `xml:"password,omitempty"`
	Email        *string    `xml:"email,omitempty"`
	Name         *string    `xml:"name,omitempty"`
	First        *string    `xml:"first,omitempty"`
	Last         *string    `xml:"last,omitempty"`
	Nick         *string    `xml:"nick,omitempty"`
	Address      *string    `xml:"address,omitempty"`
	City         *string    `xml:"city,omitempty"`
	State        *string    `xml:"state,omitempty"`
	Zip          *string    `xml:"zip,omitempty"`
	Phone        *string    `xml:"phone,omitempty"`
	URL          *string    `xml:"url,omitempty"`
	Date         *string    `xml:"date,omitempty"`
	Misc         *string    `xml:"misc,omitempty"`
	Text         *string    `xml:"text,omitempty"`
	Key          *string    `xml:"key,omitempty"`
	Registered   *struct{}  `xml:"registered,omitempty"`
	XData        *xDataForm `xml:"x,omitempty"`
	BobData      []bobData  `xml:"data,omitempty"`
}

// XEP-0004 Data Forms support
type xDataForm struct {
	XMLName      xml.Name     `xml:"x"`
	XMLNS        string       `xml:"xmlns,attr"`
	Type         string       `xml:"type,attr"`
	Title        string       `xml:"title,omitempty"`
	Instructions []string     `xml:"instructions,omitempty"`
	Fields       []xDataField `xml:"field"`
}

type xDataField struct {
	XMLName  xml.Name      `xml:"field"`
	Var      string        `xml:"var,attr,omitempty"`
	Type     string        `xml:"type,attr,omitempty"`
	Label    string        `xml:"label,attr,omitempty"`
	Required *struct{}     `xml:"required,omitempty"`
	Value    []string      `xml:"value"`
	Options  []xDataOption `xml:"option,omitempty"`
	Media    *mediaElement `xml:"media,omitempty"`
}

type xDataOption struct {
	Label string `xml:"label,attr,omitempty"`
	Value string `xml:"value"`
}

// XEP-0221 Media Element
type mediaElement struct {
	XMLName xml.Name   `xml:"urn:xmpp:media-element media"`
	Height  int        `xml:"height,attr,omitempty"`
	Width   int        `xml:"width,attr,omitempty"`
	URIs    []mediaURI `xml:"uri"`
}

type mediaURI struct {
	Type string `xml:"type,attr"`
	URI  string `xml:",chardata"`
}

// XEP-0231 Bits of Binary
type bobData struct {
	XMLName xml.Name `xml:"urn:xmpp:bob data"`
	CID     string   `xml:"cid,attr"`
	Type    string   `xml:"type,attr"`
	MaxAge  int      `xml:"max-age,attr,omitempty"`
	Data    string   `xml:",chardata"`
}

type stanzaError struct {
	XMLName   xml.Name `xml:"error"`
	Type      string   `xml:"type,attr"`
	Code      string   `xml:"code,attr,omitempty"`
	Condition string   `xml:",any"`
}

// FetchRegistrationForm connects to the server and retrieves the registration form
func FetchRegistrationForm(ctx context.Context, server string, port int) (*RegistrationForm, error) {
	if port == 0 {
		port = 5222
	}

	addr := fmt.Sprintf("%s:%d", server, port)

	// Create connection with timeout
	dialer := net.Dialer{Timeout: 30 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to server: %w", err)
	}
	defer conn.Close()

	// Set deadline for the entire operation
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	}

	// Send initial stream header
	streamHeader := fmt.Sprintf(`<?xml version='1.0'?><stream:stream to='%s' version='1.0' xmlns='jabber:client' xmlns:stream='http://etherx.jabber.org/streams'>`, server)
	if _, err := conn.Write([]byte(streamHeader)); err != nil {
		return nil, fmt.Errorf("failed to send stream header: %w", err)
	}

	decoder := xml.NewDecoder(conn)

	// Read stream response and features
	features, err := readStreamFeatures(decoder)
	if err != nil {
		return nil, fmt.Errorf("failed to read stream features: %w", err)
	}

	// Check if STARTTLS is required/available and upgrade
	if features.StartTLS != nil {
		conn, decoder, err = upgradeToTLS(conn, decoder, server)
		if err != nil {
			return nil, fmt.Errorf("TLS upgrade failed: %w", err)
		}

		// Send new stream header after TLS
		streamHeader := fmt.Sprintf(`<?xml version='1.0'?><stream:stream to='%s' version='1.0' xmlns='jabber:client' xmlns:stream='http://etherx.jabber.org/streams'>`, server)
		if _, err := conn.Write([]byte(streamHeader)); err != nil {
			return nil, fmt.Errorf("failed to send stream header after TLS: %w", err)
		}

		// Read new features
		_, err = readStreamFeatures(decoder)
		if err != nil {
			return nil, fmt.Errorf("failed to read stream features after TLS: %w", err)
		}
	}

	// Send registration query
	iq := iqStanza{
		Type: "get",
		ID:   "reg1",
		To:   server,
		Query: &registerQuery{
			XMLNS: NS,
		},
	}

	iqBytes, err := xml.Marshal(iq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal IQ: %w", err)
	}

	if _, err := conn.Write(iqBytes); err != nil {
		return nil, fmt.Errorf("failed to send registration query: %w", err)
	}

	// Read registration form response
	form, err := readRegistrationForm(decoder, server, port)
	if err != nil {
		return nil, err
	}

	// Close stream
	_, _ = conn.Write([]byte("</stream:stream>"))

	return form, nil
}

// SubmitRegistration submits the registration form to the server
func SubmitRegistration(ctx context.Context, server string, port int, fields map[string]string, isDataForm bool, formType string) (*RegistrationResult, error) {
	if port == 0 {
		port = 5222
	}

	addr := fmt.Sprintf("%s:%d", server, port)

	// Create connection with timeout
	dialer := net.Dialer{Timeout: 30 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to server: %w", err)
	}
	defer conn.Close()

	// Set deadline for the entire operation
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	}

	// Send initial stream header
	streamHeader := fmt.Sprintf(`<?xml version='1.0'?><stream:stream to='%s' version='1.0' xmlns='jabber:client' xmlns:stream='http://etherx.jabber.org/streams'>`, server)
	if _, err := conn.Write([]byte(streamHeader)); err != nil {
		return nil, fmt.Errorf("failed to send stream header: %w", err)
	}

	decoder := xml.NewDecoder(conn)

	// Read stream response and features
	features, err := readStreamFeatures(decoder)
	if err != nil {
		return nil, fmt.Errorf("failed to read stream features: %w", err)
	}

	// Check if STARTTLS is required/available and upgrade
	if features.StartTLS != nil {
		conn, decoder, err = upgradeToTLS(conn, decoder, server)
		if err != nil {
			return nil, fmt.Errorf("TLS upgrade failed: %w", err)
		}

		// Send new stream header after TLS
		streamHeader := fmt.Sprintf(`<?xml version='1.0'?><stream:stream to='%s' version='1.0' xmlns='jabber:client' xmlns:stream='http://etherx.jabber.org/streams'>`, server)
		if _, err := conn.Write([]byte(streamHeader)); err != nil {
			return nil, fmt.Errorf("failed to send stream header after TLS: %w", err)
		}

		// Read new features (discard)
		_, err = readStreamFeatures(decoder)
		if err != nil {
			return nil, fmt.Errorf("failed to read stream features after TLS: %w", err)
		}
	}

	// Build registration IQ with fields
	iq := buildRegistrationIQ(server, fields, isDataForm, formType)

	iqBytes, err := xml.Marshal(iq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal IQ: %w", err)
	}

	if _, err := conn.Write(iqBytes); err != nil {
		return nil, fmt.Errorf("failed to send registration: %w", err)
	}

	// Read registration result
	result, err := readRegistrationResult(decoder, server, fields["username"])
	if err != nil {
		return nil, err
	}

	// Close stream
	_, _ = conn.Write([]byte("</stream:stream>"))

	return result, nil
}

func readStreamFeatures(decoder *xml.Decoder) (*streamFeatures, error) {
	// Skip until we find stream:stream start element
	for {
		tok, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("error reading token: %w", err)
		}

		if se, ok := tok.(xml.StartElement); ok {
			if se.Name.Local == "stream" {
				break
			}
		}
	}

	// Read features
	for {
		tok, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("error reading token: %w", err)
		}

		if se, ok := tok.(xml.StartElement); ok {
			if se.Name.Local == "features" {
				var features streamFeatures
				if err := decoder.DecodeElement(&features, &se); err != nil {
					return nil, fmt.Errorf("error decoding features: %w", err)
				}
				return &features, nil
			}
		}
	}
}

func upgradeToTLS(conn net.Conn, decoder *xml.Decoder, server string) (net.Conn, *xml.Decoder, error) {
	// Send STARTTLS
	startTLSReq := `<starttls xmlns='urn:ietf:params:xml:ns:xmpp-tls'/>`
	if _, err := conn.Write([]byte(startTLSReq)); err != nil {
		return nil, nil, fmt.Errorf("failed to send STARTTLS: %w", err)
	}

	// Read response
	for {
		tok, err := decoder.Token()
		if err != nil {
			return nil, nil, fmt.Errorf("error reading STARTTLS response: %w", err)
		}

		if se, ok := tok.(xml.StartElement); ok {
			if se.Name.Local == "proceed" {
				break
			}
			if se.Name.Local == "failure" {
				return nil, nil, fmt.Errorf("STARTTLS failed")
			}
		}
	}

	// Upgrade to TLS
	tlsConfig := &tls.Config{
		ServerName: server,
		MinVersion: tls.VersionTLS12,
	}

	tlsConn := tls.Client(conn, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		return nil, nil, fmt.Errorf("TLS handshake failed: %w", err)
	}

	return tlsConn, xml.NewDecoder(tlsConn), nil
}

func readRegistrationForm(decoder *xml.Decoder, server string, port int) (*RegistrationForm, error) {
	for {
		tok, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				return nil, fmt.Errorf("connection closed unexpectedly")
			}
			return nil, fmt.Errorf("error reading token: %w", err)
		}

		if se, ok := tok.(xml.StartElement); ok {
			if se.Name.Local == "iq" {
				var iq iqStanza
				if err := decoder.DecodeElement(&iq, &se); err != nil {
					return nil, fmt.Errorf("error decoding IQ: %w", err)
				}

				if iq.Type == "error" {
					errMsg := "server does not support registration"
					if iq.Error != nil {
						errMsg = parseErrorCondition(iq.Error)
					}
					return nil, errors.New(errMsg)
				}

				if iq.Type == "result" && iq.Query != nil {
					return parseRegistrationQuery(iq.Query, server, port), nil
				}
			}
		}
	}
}

func parseRegistrationQuery(query *registerQuery, server string, port int) *RegistrationForm {
	form := &RegistrationForm{
		Server:       server,
		Port:         port,
		Instructions: query.Instructions,
		Fields:       []RegistrationField{},
	}

	// Check if server sent XEP-0004 Data Form
	if query.XData != nil && query.XData.XMLNS == "jabber:x:data" {
		// Build BOB data map from any <data> elements
		bobDataMap := make(map[string]bobData)
		for _, bob := range query.BobData {
			bobDataMap[bob.CID] = bob
		}
		return parseDataForm(query.XData, server, port, query.Instructions, bobDataMap)
	}

	// Legacy XEP-0077 simple fields
	addFieldIfPresent := func(name string, value *string) {
		if value != nil {
			label := fieldLabels[name]
			if label == "" {
				// Capitalize first letter manually to avoid deprecated strings.Title
				if len(name) > 0 {
					label = strings.ToUpper(name[:1]) + name[1:]
				} else {
					label = name
				}
			}
			form.Fields = append(form.Fields, RegistrationField{
				Name:     name,
				Label:    label,
				Required: name == "username" || name == "password",
				Password: passwordFields[name],
				Type:     "text-single",
			})
		}
	}

	addFieldIfPresent("username", query.Username)
	addFieldIfPresent("password", query.Password)
	addFieldIfPresent("email", query.Email)
	addFieldIfPresent("name", query.Name)
	addFieldIfPresent("first", query.First)
	addFieldIfPresent("last", query.Last)
	addFieldIfPresent("nick", query.Nick)
	addFieldIfPresent("address", query.Address)
	addFieldIfPresent("city", query.City)
	addFieldIfPresent("state", query.State)
	addFieldIfPresent("zip", query.Zip)
	addFieldIfPresent("phone", query.Phone)
	addFieldIfPresent("url", query.URL)
	addFieldIfPresent("date", query.Date)
	addFieldIfPresent("misc", query.Misc)
	addFieldIfPresent("text", query.Text)
	addFieldIfPresent("key", query.Key)

	return form
}

// parseDataForm parses XEP-0004 Data Forms
func parseDataForm(xdata *xDataForm, server string, port int, fallbackInstructions string, bobDataMap map[string]bobData) *RegistrationForm {
	form := &RegistrationForm{
		Server:     server,
		Port:       port,
		IsDataForm: true,
		Fields:     []RegistrationField{},
	}

	// Use data form instructions if available, otherwise fallback
	if len(xdata.Instructions) > 0 {
		form.Instructions = strings.Join(xdata.Instructions, "\n")
	} else if xdata.Title != "" {
		form.Instructions = xdata.Title
	} else {
		form.Instructions = fallbackInstructions
	}

	// Check if instructions contain a CAPTCHA URL
	if strings.Contains(form.Instructions, "http://") || strings.Contains(form.Instructions, "https://") {
		// Try to extract URL from instructions
		words := strings.Fields(form.Instructions)
		for _, word := range words {
			if strings.HasPrefix(word, "http://") || strings.HasPrefix(word, "https://") {
				// Clean up URL (remove trailing punctuation)
				url := strings.TrimRight(word, ".,;:!?")
				if form.Captcha == nil {
					form.Captcha = &CaptchaData{
						Type: "image",
						URL:  url,
					}
					form.RequiresCaptcha = true
				}
				break
			}
		}
	}

	for _, field := range xdata.Fields {
		// Skip FORM_TYPE field but record it
		if field.Var == "FORM_TYPE" {
			if len(field.Value) > 0 {
				form.FormType = field.Value[0]
			}
			continue
		}

		// Detect CAPTCHA challenge types per XEP-0158
		challengeType := detectChallengeType(field.Var)
		isCaptchaField := challengeType != "" ||
			strings.Contains(strings.ToLower(field.Var), "captcha") ||
			strings.Contains(strings.ToLower(field.Label), "captcha")

		if isCaptchaField || field.Media != nil {
			form.RequiresCaptcha = true

			// Initialize captcha if not already set
			if form.Captcha == nil {
				form.Captcha = &CaptchaData{
					Challenge: challengeType,
					FieldVar:  field.Var,
					Question:  field.Label,
				}
			}

			// Determine media type category from challenge type
			form.Captcha.Type = getCaptchaMediaType(challengeType, field.Media)

			// Try to extract CAPTCHA data from media element (XEP-0221)
			if field.Media != nil && len(field.Media.URIs) > 0 {
				for _, uri := range field.Media.URIs {
					// Handle different URI schemes
					if cid, ok := strings.CutPrefix(uri.URI, "cid:"); ok {
						// CID reference to BOB data (XEP-0231)
						if data, mimeType, found := lookupBOBData(cid, bobDataMap); found {
							form.Captcha.Data = data
							form.Captcha.MimeType = mimeType
						}
					} else if dataURI, ok := strings.CutPrefix(uri.URI, "data:"); ok {
						// data: URI scheme (RFC 2397) - inline base64 data
						if data, mimeType, ok := parseDataURI(dataURI); ok {
							form.Captcha.Data = data
							form.Captcha.MimeType = mimeType
						}
					} else if strings.HasPrefix(uri.URI, "http://") || strings.HasPrefix(uri.URI, "https://") {
						// HTTP(S) URL
						form.Captcha.URLs = append(form.Captcha.URLs, uri.URI)
						if form.Captcha.URL == "" {
							form.Captcha.URL = uri.URI
						}
						if form.Captcha.MimeType == "" {
							form.Captcha.MimeType = uri.Type
						}
					}
				}
			}

			// Check if URL is in the field value directly
			if form.Captcha.URL == "" && len(form.Captcha.Data) == 0 && len(field.Value) > 0 {
				val := field.Value[0]
				if strings.HasPrefix(val, "http://") || strings.HasPrefix(val, "https://") {
					form.Captcha.URL = val
					form.Captcha.URLs = append(form.Captcha.URLs, val)
				} else if dataURI, ok := strings.CutPrefix(val, "data:"); ok {
					if data, mimeType, ok := parseDataURI(dataURI); ok {
						form.Captcha.Data = data
						form.Captcha.MimeType = mimeType
					}
				}
			}

			// Check if URL is in field label (some servers do this)
			if form.Captcha.URL == "" && len(form.Captcha.Data) == 0 {
				if strings.HasPrefix(field.Label, "http://") || strings.HasPrefix(field.Label, "https://") {
					form.Captcha.URL = field.Label
					form.Captcha.URLs = append(form.Captcha.URLs, field.Label)
				}
			}

			// For QA-type CAPTCHA, use the label as the question
			if challengeType == "qa" {
				form.Captcha.Type = "qa"
				form.Captcha.Question = field.Label
			}
		}

		// Skip fixed fields (just display text)
		if field.Type == "fixed" {
			continue
		}

		// Determine label
		label := field.Label
		if label == "" {
			label = fieldLabels[field.Var]
		}
		if label == "" {
			if len(field.Var) > 0 {
				label = strings.ToUpper(field.Var[:1]) + field.Var[1:]
			} else {
				label = field.Var
			}
		}

		// Determine if password field
		isPassword := field.Type == "text-private" || passwordFields[field.Var]

		// Get default value
		defaultValue := ""
		if len(field.Value) > 0 {
			defaultValue = field.Value[0]
		}

		regField := RegistrationField{
			Name:     field.Var,
			Label:    label,
			Required: field.Required != nil,
			Password: isPassword,
			Type:     field.Type,
			Value:    defaultValue,
		}

		// Hidden fields should be preserved but not shown to user
		// We'll include them so they get submitted back
		form.Fields = append(form.Fields, regField)
	}

	return form
}

func buildRegistrationIQ(server string, fields map[string]string, isDataForm bool, formType string) iqStanza {
	query := &registerQuery{
		XMLNS: NS,
	}

	if isDataForm {
		// Build XEP-0004 Data Form response
		xdata := &xDataForm{
			XMLNS: "jabber:x:data",
			Type:  "submit",
		}

		// Add FORM_TYPE if we have one (no type attribute in submit forms per XEP-0004)
		if formType != "" {
			xdata.Fields = append(xdata.Fields, xDataField{
				Var:   "FORM_TYPE",
				Value: []string{formType},
			})
		}

		// Add all other fields
		for name, value := range fields {
			if name == "_isDataForm" || name == "_formType" {
				continue
			}
			xdata.Fields = append(xdata.Fields, xDataField{
				Var:   name,
				Value: []string{value},
			})
		}

		query.XData = xdata
	} else {
		// Legacy XEP-0077 format
		if v, ok := fields["username"]; ok {
			query.Username = &v
		}
		if v, ok := fields["password"]; ok {
			query.Password = &v
		}
		if v, ok := fields["email"]; ok && v != "" {
			query.Email = &v
		}
		if v, ok := fields["name"]; ok && v != "" {
			query.Name = &v
		}
		if v, ok := fields["first"]; ok && v != "" {
			query.First = &v
		}
		if v, ok := fields["last"]; ok && v != "" {
			query.Last = &v
		}
		if v, ok := fields["nick"]; ok && v != "" {
			query.Nick = &v
		}
		if v, ok := fields["address"]; ok && v != "" {
			query.Address = &v
		}
		if v, ok := fields["city"]; ok && v != "" {
			query.City = &v
		}
		if v, ok := fields["state"]; ok && v != "" {
			query.State = &v
		}
		if v, ok := fields["zip"]; ok && v != "" {
			query.Zip = &v
		}
		if v, ok := fields["phone"]; ok && v != "" {
			query.Phone = &v
		}
		if v, ok := fields["url"]; ok && v != "" {
			query.URL = &v
		}
		if v, ok := fields["date"]; ok && v != "" {
			query.Date = &v
		}
		if v, ok := fields["misc"]; ok && v != "" {
			query.Misc = &v
		}
		if v, ok := fields["text"]; ok && v != "" {
			query.Text = &v
		}
		if v, ok := fields["key"]; ok && v != "" {
			query.Key = &v
		}
	}

	return iqStanza{
		Type:  "set",
		ID:    "reg2",
		To:    server,
		Query: query,
	}
}

func readRegistrationResult(decoder *xml.Decoder, server, username string) (*RegistrationResult, error) {
	for {
		tok, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				return nil, fmt.Errorf("connection closed unexpectedly")
			}
			return nil, fmt.Errorf("error reading token: %w", err)
		}

		if se, ok := tok.(xml.StartElement); ok {
			if se.Name.Local == "iq" {
				var iq iqStanza
				if err := decoder.DecodeElement(&iq, &se); err != nil {
					return nil, fmt.Errorf("error decoding IQ: %w", err)
				}

				if iq.Type == "error" {
					errMsg := "registration failed"
					if iq.Error != nil {
						errMsg = parseErrorCondition(iq.Error)
					}
					return &RegistrationResult{
						Success: false,
						Error:   errMsg,
					}, nil
				}

				if iq.Type == "result" {
					jid := username + "@" + server
					return &RegistrationResult{
						Success: true,
						JID:     jid,
					}, nil
				}
			}
		}
	}
}

func parseErrorCondition(err *stanzaError) string {
	// Map common error conditions to user-friendly messages
	condition := strings.TrimSpace(err.Condition)

	switch {
	case strings.Contains(condition, "conflict"):
		return "username is already taken"
	case strings.Contains(condition, "not-acceptable"):
		return "registration information not acceptable"
	case strings.Contains(condition, "not-allowed"):
		return "registration not allowed"
	case strings.Contains(condition, "forbidden"):
		return "registration forbidden"
	case strings.Contains(condition, "service-unavailable"):
		return "server does not support registration"
	case strings.Contains(condition, "resource-constraint"):
		return "server resource limit reached"
	case strings.Contains(condition, "bad-request"):
		return "invalid registration request"
	default:
		if condition != "" {
			return condition
		}
		return "registration failed"
	}
}

// detectChallengeType identifies the XEP-0158 challenge type from field var
func detectChallengeType(fieldVar string) string {
	// XEP-0158 defined challenge types
	switch fieldVar {
	case "ocr":
		return "ocr" // Optical character recognition
	case "audio_recog":
		return "audio_recog" // Audio recognition
	case "video_recog":
		return "video_recog" // Video recognition
	case "picture_recog":
		return "picture_recog" // Picture recognition
	case "picture_q":
		return "picture_q" // Picture question
	case "speech_q":
		return "speech_q" // Speech question
	case "speech_recog":
		return "speech_recog" // Speech recognition
	case "video_q":
		return "video_q" // Video question
	case "qa":
		return "qa" // Text question/answer
	case "captcha":
		return "ocr" // Generic captcha, assume OCR
	default:
		return ""
	}
}

// getCaptchaMediaType determines the media category (image/audio/video/qa) from challenge type
func getCaptchaMediaType(challengeType string, media *mediaElement) string {
	// First check challenge type
	switch challengeType {
	case "audio_recog", "speech_q", "speech_recog":
		return "audio"
	case "video_recog", "video_q":
		return "video"
	case "qa":
		return "qa"
	case "ocr", "picture_recog", "picture_q", "captcha":
		return "image"
	}

	// Fall back to checking media MIME type
	if media != nil && len(media.URIs) > 0 {
		for _, uri := range media.URIs {
			mimeType := strings.ToLower(uri.Type)
			if strings.HasPrefix(mimeType, "audio/") {
				return "audio"
			}
			if strings.HasPrefix(mimeType, "video/") {
				return "video"
			}
			if strings.HasPrefix(mimeType, "image/") {
				return "image"
			}
		}
	}

	return "image" // Default to image
}

// lookupBOBData finds BOB data by CID with normalized matching
func lookupBOBData(cid string, bobDataMap map[string]bobData) ([]byte, string, bool) {
	// Try exact match first
	if bob, ok := bobDataMap[cid]; ok {
		return decodeBOBData(bob)
	}

	// Try with cid: prefix
	if bob, ok := bobDataMap["cid:"+cid]; ok {
		return decodeBOBData(bob)
	}

	// Try stripping cid: from map keys
	for k, bob := range bobDataMap {
		if strings.TrimPrefix(k, "cid:") == cid {
			return decodeBOBData(bob)
		}
	}

	// Try matching just the hash part (after + and before @)
	// CID format: sha1+hash@bob.xmpp.org
	cidHash := extractCIDHash(cid)
	if cidHash != "" {
		for k, bob := range bobDataMap {
			if extractCIDHash(k) == cidHash || extractCIDHash(strings.TrimPrefix(k, "cid:")) == cidHash {
				return decodeBOBData(bob)
			}
		}
	}

	return nil, "", false
}

// extractCIDHash extracts the hash portion from a CID
// CID format: algo+hash@domain or just hash@domain
func extractCIDHash(cid string) string {
	// Remove cid: prefix if present
	cid = strings.TrimPrefix(cid, "cid:")

	// Find the hash part (after + if present, before @)
	atIdx := strings.Index(cid, "@")
	if atIdx > 0 {
		cid = cid[:atIdx]
	}

	plusIdx := strings.Index(cid, "+")
	if plusIdx >= 0 {
		return cid[plusIdx+1:]
	}

	return cid
}

// decodeBOBData decodes base64 BOB data
func decodeBOBData(bob bobData) ([]byte, string, bool) {
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(bob.Data))
	if err != nil {
		return nil, "", false
	}
	return decoded, bob.Type, true
}

// parseDataURI parses a data: URI (RFC 2397)
// Format: [mediatype][;base64],data
func parseDataURI(dataURI string) ([]byte, string, bool) {
	// Find the comma separating metadata from data
	commaIdx := strings.Index(dataURI, ",")
	if commaIdx < 0 {
		return nil, "", false
	}

	metadata := dataURI[:commaIdx]
	data := dataURI[commaIdx+1:]

	// Parse metadata
	mimeType := "text/plain" // Default per RFC 2397
	isBase64 := false

	parts := strings.Split(metadata, ";")
	for i, part := range parts {
		if i == 0 && part != "" {
			mimeType = part
		} else if part == "base64" {
			isBase64 = true
		}
	}

	// Decode data
	var decoded []byte
	var err error
	if isBase64 {
		decoded, err = base64.StdEncoding.DecodeString(data)
		if err != nil {
			// Try URL-safe base64
			decoded, err = base64.URLEncoding.DecodeString(data)
			if err != nil {
				return nil, "", false
			}
		}
	} else {
		// URL-encoded data
		decoded = []byte(data) // Simplified - proper impl would URL-decode
	}

	return decoded, mimeType, true
}
