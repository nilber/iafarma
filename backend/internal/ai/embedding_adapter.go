package ai

// EmbeddingServiceAdapter is a simple adapter that will be initialized with the actual service
type EmbeddingServiceAdapter struct {
	searchProductsFunc             func(query, tenantID string, limit int) ([]ProductSearchResult, error)
	searchConversationsFunc        func(tenantID, customerID, query string, limit int) ([]ConversationSearchResult, error)
	searchConversationsWithAgeFunc func(tenantID, customerID, query string, limit int, maxAgeHours int) ([]ConversationSearchResult, error)
	storeConversationFunc          func(tenantID, customerID string, entry ConversationEntry) error
	cleanupOldConversationsFunc    func(tenantID, customerID string, maxAgeHours int) (int, error)
}

// NewEmbeddingServiceAdapterWithFuncs creates a new adapter with function pointers
func NewEmbeddingServiceAdapterWithFuncs(
	searchProducts func(query, tenantID string, limit int) ([]ProductSearchResult, error),
	searchConversations func(tenantID, customerID, query string, limit int) ([]ConversationSearchResult, error),
	searchConversationsWithAge func(tenantID, customerID, query string, limit int, maxAgeHours int) ([]ConversationSearchResult, error),
	storeConversation func(tenantID, customerID string, entry ConversationEntry) error,
	cleanupOldConversations func(tenantID, customerID string, maxAgeHours int) (int, error),
) EmbeddingServiceInterface {
	return &EmbeddingServiceAdapter{
		searchProductsFunc:             searchProducts,
		searchConversationsFunc:        searchConversations,
		searchConversationsWithAgeFunc: searchConversationsWithAge,
		storeConversationFunc:          storeConversation,
		cleanupOldConversationsFunc:    cleanupOldConversations,
	}
}

// SearchSimilarProducts adapts the call to the underlying service
func (a *EmbeddingServiceAdapter) SearchSimilarProducts(query, tenantID string, limit int) ([]ProductSearchResult, error) {
	if a.searchProductsFunc != nil {
		return a.searchProductsFunc(query, tenantID, limit)
	}
	return nil, nil
}

// SearchConversations adapts the call to the underlying service
func (a *EmbeddingServiceAdapter) SearchConversations(tenantID, customerID, query string, limit int) ([]ConversationSearchResult, error) {
	if a.searchConversationsFunc != nil {
		return a.searchConversationsFunc(tenantID, customerID, query, limit)
	}
	return nil, nil
}

// SearchConversationsWithMaxAge adapts the call to the underlying service
func (a *EmbeddingServiceAdapter) SearchConversationsWithMaxAge(tenantID, customerID, query string, limit int, maxAgeHours int) ([]ConversationSearchResult, error) {
	if a.searchConversationsWithAgeFunc != nil {
		return a.searchConversationsWithAgeFunc(tenantID, customerID, query, limit, maxAgeHours)
	}
	return nil, nil
}

// StoreConversation adapts the call to the underlying service
func (a *EmbeddingServiceAdapter) StoreConversation(tenantID, customerID string, entry ConversationEntry) error {
	if a.storeConversationFunc != nil {
		return a.storeConversationFunc(tenantID, customerID, entry)
	}
	return nil
}

// CleanupOldConversations adapts the call to the underlying service
func (a *EmbeddingServiceAdapter) CleanupOldConversations(tenantID, customerID string, maxAgeHours int) (int, error) {
	if a.cleanupOldConversationsFunc != nil {
		return a.cleanupOldConversationsFunc(tenantID, customerID, maxAgeHours)
	}
	return 0, nil
}
