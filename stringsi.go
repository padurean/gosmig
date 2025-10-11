package gosmig

func singularOrPlural(singular string, n int) string {
	if n == 1 {
		return singular
	}
	return singular + "s"
}
