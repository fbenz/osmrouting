
package mm

func Open(path string, p interface{}) error {
	m, err  := sys_open(path)
	if err != nil {
		return err
	}
	reflect_set(p, m)
	return nil
}

func Create(path string, size int, p interface{}) error {
	rsize := size * reflect_elem_size(p)
	m, err  := sys_create(path, rsize)
	if err != nil {
		return err
	}
	reflect_set(p, m)
	return nil
}

func Sync(p interface{}) error {
	return sys_sync(reflect_get(p))
}

func Close(p interface{}) error {
	b := reflect_get(p)
	err := sys_sync(b)
	if err != nil {
		return err
	}
	err = sys_close(b)
	if err != nil {
		return err
	}
	reflect_set(p, nil)
	return nil
}
