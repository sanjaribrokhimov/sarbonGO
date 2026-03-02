// Пакет test: один тест-указатель на описание flow.
// Какой flow проверяется: см. test/FLOWS.md (Driver и Freelance Dispatcher).
// Запуск всех flow: go test -v ./test/...
package test

import (
	"os"
	"testing"
)

// TestFlowsDocumentation заставляет go test ./test/ видеть пакет test и напоминает про FLOWS.md.
// Реальные тесты Driver и Freelance Dispatcher находятся в ./test/driver/ и ./test/dispatcher/.
func TestFlowsDocumentation(t *testing.T) {
	if os.Getenv("SKIP_FLOWS_DOC") != "" {
		t.Skip("SKIP_FLOWS_DOC set")
	}
	t.Log("Driver и Freelance Dispatcher flows: описание и список проверок — см. test/FLOWS.md")
	t.Log("Запуск: go test -v ./test/driver/ и go test -v ./test/dispatcher/ или go test -v ./test/...")
}
