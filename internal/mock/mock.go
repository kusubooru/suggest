package mock

import "github.com/kusubooru/teian/teian"

type SuggestionStore struct {
	CreateFn      func(username string, sugg *teian.Suggestion) error
	CreateInvoked bool

	OfUserFn      func(username string) ([]teian.Suggestion, error)
	OfUserInvoked bool

	AllFn      func() ([]teian.Suggestion, error)
	AllInvoked bool

	DeleteFn      func(username string, id uint64) error
	DeleteInvoked bool

	CheckQuotaFn      func(username string, n teian.Quota) (teian.Quota, error)
	CheckQuotaInvoked bool
}

func (s *SuggestionStore) Create(username string, sugg *teian.Suggestion) error {
	s.CreateInvoked = true
	return s.CreateFn(username, sugg)
}
func (s *SuggestionStore) OfUser(username string) ([]teian.Suggestion, error) {
	s.OfUserInvoked = true
	return s.OfUserFn(username)
}
func (s *SuggestionStore) All() ([]teian.Suggestion, error) {
	s.AllInvoked = true
	return s.AllFn()
}
func (s *SuggestionStore) Delete(username string, id uint64) error {
	s.DeleteInvoked = true
	return s.DeleteFn(username, id)
}
func (s *SuggestionStore) CheckQuota(username string, n teian.Quota) (teian.Quota, error) {
	s.CheckQuotaInvoked = true
	return s.CheckQuotaFn(username, n)
}
