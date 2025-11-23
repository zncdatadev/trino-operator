package common

type Role string

const (
	RoleCoordinator Role = "coordinator"
	RoleWorker      Role = "worker"
)

const (
	HttpScheme  = "http"
	HttpsScheme = "https"
)
