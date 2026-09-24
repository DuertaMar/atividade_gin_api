package main

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Struct
type Aluno struct {
	Nome               string `json:"nome"`
	Matricula          int    `json:"matricula"`
	EmailUniversitario string `json:"email_universitario"`
}

// Request
type CriarAlunoRequest struct {
	Nome         string `json:"nome"`
	Matricula    int    `json:"matricula"`
	UsuarioEmail string `json:"usuario_email"`
}

type AtualizarAlunoRequest struct {
	Nome string `json:"nome"`
}

// Controller
type AlunoController struct {
	DB *Database
}

func NovoAlunoController(db *Database) *AlunoController {
	return &AlunoController{
		DB: db,
	}
}

// End Points
// POST /api/v1/alunos
func (ctrl *AlunoController) Criar(c *gin.Context) {
	var req CriarAlunoRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "Dados de requisição inválidos."})
		return
	}

	if strings.TrimSpace(req.Nome) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "O nome do aluno é obrigatório."})
		return
	}
	if req.Matricula <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "A matrícula deve ser maior que zero."})
		return
	}
	if strings.TrimSpace(req.UsuarioEmail) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "O usuário do e-mail é obrigatório."})
		return
	}

	if MatriculaJaExiste(req.Matricula, ctrl.DB.Alunos) {
		c.JSON(http.StatusConflict, gin.H{"erro": "Já existe um aluno com essa matrícula."})
		return
	}

	if strings.Contains(req.UsuarioEmail, "@") {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "Digite apenas o usuário do e-mail (sem o @)."})
		return
	}

	emailGerado := req.UsuarioEmail + "@edu.unifil.br"
	novoAluno := CadastrarAluno(req.Nome, req.Matricula, emailGerado)

	ctrl.DB.Alunos = append(ctrl.DB.Alunos, novoAluno)

	c.JSON(http.StatusCreated, gin.H{
		"mensagem": "Aluno cadastrado com sucesso!",
		"aluno":    novoAluno,
	})
}

// GET /api/v1/alunos
func (ctrl *AlunoController) Listar(c *gin.Context) {
	if len(ctrl.DB.Alunos) == 0 {
		c.JSON(http.StatusOK, gin.H{"mensagem": "Nenhum aluno cadastrado.", "alunos": []Aluno{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"alunos": ctrl.DB.Alunos})
}

// GET /api/v1/alunos/:matricula
func (ctrl *AlunoController) BuscarPorMatricula(c *gin.Context) {
	matriculaParam := c.Param("matricula")
	matricula, err := strconv.Atoi(matriculaParam)
	if err != nil || matricula <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "Matrícula inválida."})
		return
	}

	indice := BuscarIndiceAluno(matricula, ctrl.DB.Alunos)
	if indice == -1 {
		c.JSON(http.StatusNotFound, gin.H{"erro": "Aluno não encontrado."})
		return
	}

	aluno := ctrl.DB.Alunos[indice]

	var turmasMatriculadas []gin.H
	for _, t := range ctrl.DB.Turmas {
		for _, a := range t.Alunos {
			if a.Matricula == matricula {
				turmasMatriculadas = append(turmasMatriculadas, gin.H{
					"turma":      t.Nome,
					"disciplina": t.Disciplina,
					"professor":  t.Professor,
				})
				break
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"aluno":  aluno,
		"turmas": turmasMatriculadas,
	})
}

// PUT /api/v1/alunos/:matricula
func (ctrl *AlunoController) Atualizar(c *gin.Context) {
	matriculaParam := c.Param("matricula")
	matricula, err := strconv.Atoi(matriculaParam)
	if err != nil || matricula <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "Matrícula inválida."})
		return
	}

	var req AtualizarAlunoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "Dados de requisição inválidos."})
		return
	}

	if strings.TrimSpace(req.Nome) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "O nome do aluno é obrigatório."})
		return
	}

	sucesso := AtualizarAluno(matricula, req.Nome, ctrl.DB.Alunos, ctrl.DB.Turmas)
	if !sucesso {
		c.JSON(http.StatusNotFound, gin.H{"erro": "Aluno não encontrado."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"mensagem": "Nome atualizado com sucesso no sistema e nas turmas."})
}

// DELETE /api/v1/alunos/:matricula
func (ctrl *AlunoController) Deletar(c *gin.Context) {
	matriculaParam := c.Param("matricula")
	matricula, err := strconv.Atoi(matriculaParam)
	if err != nil || matricula <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "Matrícula inválida."})
		return
	}

	var sucesso bool
	ctrl.DB.Alunos, sucesso = DeletarAluno(matricula, ctrl.DB.Alunos, ctrl.DB.Turmas)

	if !sucesso {
		c.JSON(http.StatusNotFound, gin.H{"erro": "Aluno não encontrado."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"mensagem": "Aluno deletado com sucesso do sistema e das turmas."})
}

// Funções
func CadastrarAluno(nome string, matricula int, email string) Aluno {
	return Aluno{
		Nome:               nome,
		Matricula:          matricula,
		EmailUniversitario: email,
	}
}

func AtualizarAluno(matricula int, novoNome string, alunos []Aluno, turmas []Turma) bool {
	indice := BuscarIndiceAluno(matricula, alunos)
	if indice == -1 {
		return false
	}

	alunos[indice].Nome = novoNome
	for i := range turmas {
		idx := BuscarIndiceAluno(matricula, turmas[i].Alunos)
		if idx != -1 {
			turmas[i].Alunos[idx].Nome = novoNome
		}
	}

	return true
}

func DeletarAluno(matricula int, alunos []Aluno, turmas []Turma) ([]Aluno, bool) {
	indice := BuscarIndiceAluno(matricula, alunos)
	if indice == -1 {
		return alunos, false
	}
	alunos = append(alunos[:indice], alunos[indice+1:]...)

	for i := range turmas {
		idx := BuscarIndiceAluno(matricula, turmas[i].Alunos)
		if idx != -1 {
			turmas[i].Alunos = append(turmas[i].Alunos[:idx], turmas[i].Alunos[idx+1:]...)
			turmas[i].QntAluno--
		}
	}

	return alunos, true
}

func MatriculaJaExiste(matricula int, alunos []Aluno) bool {
	for _, aluno := range alunos {
		if aluno.Matricula == matricula {
			return true
		}
	}
	return false
}

func BuscarIndiceAluno(matricula int, alunos []Aluno) int {
	for i, aluno := range alunos {
		if aluno.Matricula == matricula {
			return i
		}
	}
	return -1
}
