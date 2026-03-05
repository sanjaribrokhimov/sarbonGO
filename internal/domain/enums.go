package domain

type RegistrationStep string

const (
	StepNameOferta    RegistrationStep = "name-oferta"
	StepGeoPush       RegistrationStep = "geo-push"
	StepTransportType RegistrationStep = "transport-type"
	StepCompleted     RegistrationStep = "completed" // после отправки KYC регистрация считается завершённой
)

type RegistrationStatus string

const (
	StatusStart RegistrationStatus = "start"
	StatusBasic RegistrationStatus = "basic"
	StatusFull  RegistrationStatus = "full"
)

type DriverType string

const (
	DriverTypeCompany    DriverType = "company"
	DriverTypeFreelancer DriverType = "freelancer"
	DriverTypeDriver     DriverType = "driver"
)

