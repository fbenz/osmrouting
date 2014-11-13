/*
 * Copyright 2014 Florian Benz, Steven Schäfer, Bernhard Schommer
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */


package mm

func Open(path string, p interface{}) error {
	m, err := sys_open(path)
	if err != nil {
		return err
	}
	ProfileAllocate(len(m))
	reflect_set(p, m)
	return nil
}

func Create(path string, size int, p interface{}) error {
	rsize := size * reflect_elem_size(p)
	m, err  := sys_create(path, rsize)
	if err != nil {
		return err
	}
	ProfileAllocate(len(m))
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
	ProfileFree(len(b))
	if err := sys_close(b); err != nil {
		return err
	}
	reflect_set(p, nil)
	return nil
}
