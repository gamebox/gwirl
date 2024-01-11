package todo

import "log"

type Todo struct {
	Id        int
	Text      string
	Completed bool
	Selected  bool
	Editing   bool
}

func NewTodo(id int, text string) Todo {
	return Todo{Id: id, Text: text}
}

type TodoList struct {
	todos  []Todo
	nextId int
}

func NewTodoList() TodoList {
	return TodoList{[]Todo{}, 0}
}

func (tl *TodoList) AddTodo(text string) Todo {
	newTodo := NewTodo(tl.nextId, text)
	tl.nextId += 1
	tl.todos = append(tl.todos, newTodo)
	return newTodo
}

func (tl *TodoList) GetByFilter(filter Filter) []Todo {
	if filter.IsAll() {
		return tl.todos
	}
	list := make([]Todo, 0, len(tl.todos))
	for _, todo := range tl.todos {
		if filter.IsCompleted() && todo.Completed {
			list = append(list, todo)
			continue
		}
		if filter.IsActive() && !todo.Completed {
			list = append(list, todo)
			continue
		}
	}
	return list
}

func (tl *TodoList) TodoById(id int) *Todo {
	for _, todo := range tl.todos {
		if todo.Id == id {
			return &todo
		}
	}
	return nil
}

func (tl *TodoList) UpdateText(id int, text string) {
	for i, todo := range tl.todos {
		if todo.Id == id {
			tl.todos[i].Text = text
			return
		}
	}
}

func (tl *TodoList) ToggleEditing(id int) {
	for i, todo := range tl.todos {
		if todo.Id == id {
			log.Printf("Toggling edit for todo %d to %t", id, !todo.Editing)
			tl.todos[i].Editing = !todo.Editing
			return
		}
	}
}

func (tl *TodoList) ToggleSelected(id int) {
	for i, todo := range tl.todos {
		if todo.Id == id {
			tl.todos[i].Selected = !todo.Selected
			return
		}
	}
}

func (tl *TodoList) ToggleCompleted(id int) {
	for i, todo := range tl.todos {
		if todo.Id == id {
			tl.todos[i].Completed = !todo.Completed
			return
		}
	}
}

func (tl *TodoList) ToggleAllSelected(selected bool) {
	for i := range tl.todos {
		tl.todos[i].Selected = selected
	}
}

func (tl *TodoList) ClearCompleted() {
	list := make([]Todo, 0, len(tl.todos))
	for _, todo := range tl.todos {
		if todo.Completed {
			continue
		}
		list = append(list, todo)
	}
	tl.todos = list
}

func (tl *TodoList) RemoveTodo(id int) {
	list := make([]Todo, 0, len(tl.todos))
	for _, todo := range tl.todos {
		if todo.Id == id {
			continue
		}
		list = append(list, todo)
	}
	tl.todos = list
}

func (tl *TodoList) CompletedRemain() bool {
	for _, todo := range tl.todos {
		if todo.Completed {
			return true
		}
	}
	return false
}

type filter int

const (
	filterAll filter = iota
	filterActive
	filterCompleted
)

type Filter struct {
	filter filter
}

func DefaultFilter() Filter {
	return Filter{filterAll}
}

func ParseFilter(raw string) Filter {
	switch raw {
	case "all":
		return Filter{filterAll}
	case "active":
		return Filter{filterActive}
	case "completed":
		return Filter{filterCompleted}

	default:
		return Filter{filterAll}
	}
}

func (f Filter) IsAll() bool {
	return f.filter == filterAll
}

func (f Filter) IsActive() bool {
	return f.filter == filterActive
}

func (f Filter) IsCompleted() bool {
	return f.filter == filterCompleted
}
