package tui

import (
	"net/url"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
)

func TestSearchHit_Item(t *testing.T) {
	h := searchHit{title: "Accept a payment", url: "/payments/accept-a-payment"}
	assert.Equal(t, "Accept a payment", h.FilterValue())
	assert.Equal(t, "Accept a payment", h.Title())
	assert.Equal(t, "/payments/accept-a-payment", h.Description())
}

func TestSearchHit_Description_StripsDocsOrigin(t *testing.T) {
	h := searchHit{title: "Payments", url: "https://docs.stripe.com/payments"}
	assert.Equal(t, "/payments", h.Description())
}

func TestSearchHit_Description_PreservesExternalURL(t *testing.T) {
	h := searchHit{title: "GitHub", url: "https://github.com/stripe"}
	assert.Equal(t, "https://github.com/stripe", h.Description())
}

func TestReference_Item(t *testing.T) {
	u := &url.URL{Path: "/payments/accept-a-payment"}
	r := reference{title: "Accept a payment", url: u}
	assert.Equal(t, "Accept a payment /payments/accept-a-payment", r.FilterValue())
	assert.Equal(t, "Accept a payment", r.Title())
	assert.Equal(t, "/payments/accept-a-payment", r.Description())
}

func TestReference_Description_DocsStripeComDropsHost(t *testing.T) {
	u := mustParseURL("https://docs.stripe.com/get-started/use-cases")
	r := reference{url: u}
	assert.Equal(t, "/get-started/use-cases", r.Description())
}

func TestReference_Description_ExternalKeepsHostDropsScheme(t *testing.T) {
	u := mustParseURL("https://stripe.com/blog/post")
	r := reference{url: u, external: true}
	assert.Equal(t, "stripe.com/blog/post", r.Description())
}

func TestReference_Description_RelativePassesThrough(t *testing.T) {
	u := mustParseURL("/relative/path")
	r := reference{url: u}
	assert.Equal(t, "/relative/path", r.Description())
}

func TestReference_FilterValue_UsesPathNotDomain(t *testing.T) {
	u := mustParseURL("https://docs.stripe.com/payments/accept-a-payment")
	r := reference{title: "Accept a payment", url: u}
	assert.Equal(t, "Accept a payment /payments/accept-a-payment", r.FilterValue())
}

func mustParseURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

func TestSyncKeyMap_ReferenceModeLabel_IsView(t *testing.T) {
	u := mustParseURL("/payments")
	doc := docWithReferences(u)
	p := newPalette(Page{}, doc, nil)
	p.Open()
	p.Model, _ = p.Update(tea.KeyPressMsg{Code: '@', Text: "@"})
	p.syncKeyMap()
	assert.Equal(t, "view", p.KeyMap.Execute.Help().Desc)
}

func TestSyncKeyMap_SearchMode_LabelIsView(t *testing.T) {
	p := newPalette(Page{Content: []byte("# Test")}, nil, nil)
	// Empty input activates the catch-all search mode.
	p.syncKeyMap()
	assert.Equal(t, "view", p.KeyMap.Execute.Help().Desc)
}

func TestSyncKeyMap_CommandsMode_LabelIsExecute(t *testing.T) {
	p := newPalette(Page{Content: []byte("# Test")}, nil, nil)
	p.Open()
	p.Model, _ = p.Update(tea.KeyPressMsg{Code: '>', Text: ">"})
	p.syncKeyMap()
	assert.Equal(t, "execute", p.KeyMap.Execute.Help().Desc)
}

func TestSyncKeyMap_DisabledWhenNoSelection(t *testing.T) {
	p := newPalette(Page{Content: []byte("# Test")}, nil, nil)
	// No items in search mode with empty query → nothing selected.
	p.syncKeyMap()
	assert.False(t, p.KeyMap.Execute.Enabled())
}

func TestSyncKeyMap_EnabledWhenItemSelected(t *testing.T) {
	p := newPalette(Page{Content: []byte("# Test")}, nil, nil)
	p.Open()
	// ">" activates commands mode; empty query returns all commands.
	p.Model, _ = p.Update(tea.KeyPressMsg{Code: '>', Text: ">"})
	p.syncKeyMap()
	assert.True(t, p.KeyMap.Execute.Enabled())
}
