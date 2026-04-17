package gwservice

type genericAdapter struct{}

func (a *genericAdapter) Start(f *Facade, _ ServiceStatusReport) error {
	s, err := f.controlService()
	if err != nil {
		return &CommandError{Code: ExitServiceCommandFail, Message: err.Error()}
	}
	if err := s.Start(); err != nil {
		return &CommandError{Code: ExitServiceCommandFail, Message: err.Error()}
	}
	return nil
}

func (a *genericAdapter) Stop(f *Facade, _ ServiceStatusReport) error {
	s, err := f.controlService()
	if err != nil {
		return &CommandError{Code: ExitServiceCommandFail, Message: err.Error()}
	}
	if err := s.Stop(); err != nil {
		return &CommandError{Code: ExitServiceCommandFail, Message: err.Error()}
	}
	return nil
}

func (a *genericAdapter) Restart(f *Facade, _ ServiceStatusReport) error {
	s, err := f.controlService()
	if err != nil {
		return &CommandError{Code: ExitServiceCommandFail, Message: err.Error()}
	}
	if err := s.Restart(); err != nil {
		return &CommandError{Code: ExitServiceCommandFail, Message: err.Error()}
	}
	return nil
}
