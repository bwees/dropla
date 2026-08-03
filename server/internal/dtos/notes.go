package dtos

type CreateNoteDto struct {
	Title string `json:"title" validate:"required"`
	Body  string `json:"body"`
}
