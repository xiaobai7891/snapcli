package annotate

import "image"

// History 撤销/重做管理器
type History struct {
	annotations []Annotation // 当前标注列表
	undoStack   [][]Annotation // 撤销栈（保存之前的状态快照）
	redoStack   [][]Annotation // 重做栈
	maxHistory  int
}

// NewHistory 创建历史记录管理器
func NewHistory(maxHistory int) *History {
	if maxHistory <= 0 {
		maxHistory = 50
	}
	return &History{
		annotations: make([]Annotation, 0),
		undoStack:   make([][]Annotation, 0),
		redoStack:   make([][]Annotation, 0),
		maxHistory:  maxHistory,
	}
}

// snapshot 创建当前状态快照
func (h *History) snapshot() []Annotation {
	s := make([]Annotation, len(h.annotations))
	for i, a := range h.annotations {
		s[i] = a
		// 深拷贝 Points 切片
		s[i].Points = make([]image.Point, len(a.Points))
		copy(s[i].Points, a.Points)
	}
	return s
}

// AddAnnotation 添加一个标注（保存撤销点）
func (h *History) AddAnnotation(a Annotation) {
	// 保存当前状态到撤销栈
	h.undoStack = append(h.undoStack, h.snapshot())
	if len(h.undoStack) > h.maxHistory {
		h.undoStack = h.undoStack[1:]
	}

	// 清空重做栈（新操作后重做无效）
	h.redoStack = h.redoStack[:0]

	// 添加标注
	h.annotations = append(h.annotations, a)
}

// Undo 撤销上一步操作，返回是否成功
func (h *History) Undo() bool {
	if len(h.undoStack) == 0 {
		return false
	}

	// 保存当前状态到重做栈
	h.redoStack = append(h.redoStack, h.snapshot())

	// 恢复到上一个状态
	last := h.undoStack[len(h.undoStack)-1]
	h.undoStack = h.undoStack[:len(h.undoStack)-1]
	h.annotations = last

	return true
}

// Redo 重做上一步撤销，返回是否成功
func (h *History) Redo() bool {
	if len(h.redoStack) == 0 {
		return false
	}

	// 保存当前状态到撤销栈
	h.undoStack = append(h.undoStack, h.snapshot())

	// 恢复到下一个状态
	last := h.redoStack[len(h.redoStack)-1]
	h.redoStack = h.redoStack[:len(h.redoStack)-1]
	h.annotations = last

	return true
}

// GetAnnotations 获取当前所有标注
func (h *History) GetAnnotations() []Annotation {
	return h.annotations
}

// CanUndo 是否可以撤销
func (h *History) CanUndo() bool {
	return len(h.undoStack) > 0
}

// CanRedo 是否可以重做
func (h *History) CanRedo() bool {
	return len(h.redoStack) > 0
}

// Clear 清空所有历史
func (h *History) Clear() {
	h.annotations = h.annotations[:0]
	h.undoStack = h.undoStack[:0]
	h.redoStack = h.redoStack[:0]
}
