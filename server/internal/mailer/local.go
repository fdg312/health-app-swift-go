package mailer

import "log"

type LocalSender struct {
	logger *log.Logger
}

func NewLocalSender(logger *log.Logger) *LocalSender {
	if logger == nil {
		logger = log.Default()
	}
	return &LocalSender{logger: logger}
}

func (s *LocalSender) Send(to, subject, textBody string) error {
	s.logger.Printf("mailer.local: to=%s subject=%q body=%q", to, subject, textBody)
	return nil
}
