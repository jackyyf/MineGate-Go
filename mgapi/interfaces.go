package mgapi

type MGConfig interface {
	Get(string) (interface{}, error)
	GetString(string) (string, error)
	GetByteString(string) ([]byte, error)
	GetInt(string) (int, error)
	GetUint(string) (uint, error)
	GetInt64(string) (int64, error)
	GetUint64(string) (uint64, error)
	GetArray(string) ([]interface{}, error)
	GetMap(string) (map[string]interface{}, error)
}

/*

func GetConfig() (config MGConfig) {
	return config_data
}

*/
