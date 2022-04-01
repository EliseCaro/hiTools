package job

type TaskFunc func(w *Pool, params ...interface{}) Flag
type Flag int64

type Task struct {
	taskId int
	f      TaskFunc
	params []interface{}
}

func (t *Task) ExecuteWork(w *Pool) Flag {
	return t.f(w, t.params...)
}
