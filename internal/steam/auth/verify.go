package auth

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// IsAuthenticated проверяет по HTML-странице Steam, что сессия активна.
// Ищет элементы навигации аккаунта, которые присутствуют у любого залогиненного пользователя.
func IsAuthenticated(html []byte) (bool, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(html)))
	if err != nil {
		return false, err
	}

	// #account_pulldown — выпадашка с именем аккаунта, есть у всех авторизованных
	// независимо от наличия/отсутствия кошелька
	if doc.Find("#account_pulldown").Length() > 0 {
		return true, nil
	}

	return false, nil
}
