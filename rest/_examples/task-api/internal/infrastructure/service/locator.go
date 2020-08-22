package service

type Locator struct {
	TaskCreatorProvider
	TaskUpdaterProvider
	TaskFinderProvider
	TaskCloserProvider
}
