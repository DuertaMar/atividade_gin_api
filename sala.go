package main

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Struct
type Sala struct {
	ID         int        `json:"id"`
	Nome       string     `json:"nome"`
	Capacidade int        `json:"capacidade"`
	Recursos   []Recursos `json:"recursos"`
	Ativa      bool       `json:"ativa"`
}

type Recursos string

const (
	ArCondicionado Recursos = "Ar-Condicionado"
	TV             Recursos = "TV"
	Computador     Recursos = "Computador"
	Notebook       Recursos = "Notebook"
	Projetor       Recursos = "Projetor"
	AparelhoDeSom  Recursos = "Aparelho de som"
)

// Requests
type CriarSalaRequest struct {
	Nome       string     `json:"nome" binding:"required"`
	Capacidade int        `json:"capacidade"`
	Recursos   []Recursos `json:"recursos"`
}

type AtualizarSalaRequest struct {
	Capacidade int        `json:"capacidade"`
	Recursos   []Recursos `json:"recursos"`
}

// Controller
type SalaController struct {
	DB *Database
}

func NovoSalaController(db *Database) *SalaController {
	return &SalaController{
		DB: db,
	}
}

// End Points
// POST /api/v1/salas
func (ctrl *SalaController) Criar(c *gin.Context) {
	var req CriarSalaRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "Dados inválidos: " + err.Error()})
		return
	}

	if req.Capacidade <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "A capacidade da sala deve ser maior que zero."})
		return
	}

	if strings.TrimSpace(req.Nome) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "O nome da sala é obrigatório."})
		return
	}

	if NomeDeSalaJaExiste(req.Nome, ctrl.DB.Salas) {
		c.JSON(http.StatusConflict, gin.H{"erro": "Já existe uma sala com esse nome."})
		return
	}

	novoID := GerarIdSala(ctrl.DB.Salas)
	novaSala := CadastrarSala(novoID, req.Nome, req.Capacidade, req.Recursos)

	ctrl.DB.Salas = append(ctrl.DB.Salas, novaSala)

	c.JSON(http.StatusCreated, gin.H{
		"mensagem": "Sala cadastrada com sucesso!",
		"sala":     novaSala,
	})
}

// GET /api/v1/salas
func (ctrl *SalaController) Listar(c *gin.Context) {
	if len(ctrl.DB.Salas) == 0 {
		c.JSON(http.StatusOK, gin.H{"mensagem": "Nenhuma sala cadastrada.", "salas": []Sala{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"salas": ctrl.DB.Salas})
}

// PUT /api/v1/salas/:id
func (ctrl *SalaController) Atualizar(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "ID inválido."})
		return
	}

	var req AtualizarSalaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "Dados inválidos: " + err.Error()})
		return
	}
	if req.Capacidade <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "A capacidade da sala deve ser maior que zero."})
		return
	}

	indice := BuscarIndiceSala(id, ctrl.DB.Salas)
	if indice == -1 {
		c.JSON(http.StatusNotFound, gin.H{"erro": "Sala não encontrada."})
		return
	}

	if !PodeReduzirCapacidadeDaSala(id, req.Capacidade, ctrl.DB.Turmas) {
		c.JSON(http.StatusConflict, gin.H{
			"erro": "A nova capacidade é menor do que a quantidade de alunos em uma das turmas que já agendaram esta sala.",
		})
		return
	}

	AtualizarSalaDados(id, req.Capacidade, req.Recursos, ctrl.DB)

	c.JSON(http.StatusOK, gin.H{"mensagem": "Sala atualizada com sucesso no sistema e na agenda das turmas."})
}

// PATCH /api/v1/salas/:id/inativar
func (ctrl *SalaController) Inativar(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "ID inválido."})
		return
	}

	indice := BuscarIndiceSala(id, ctrl.DB.Salas)
	if indice == -1 {
		c.JSON(http.StatusNotFound, gin.H{"erro": "Sala não encontrada."})
		return
	}

	ctrl.DB.Salas[indice].Ativa = false
	c.JSON(http.StatusOK, gin.H{"mensagem": "Sala inativada com sucesso! O histórico foi mantido, mas ela não receberá novas alocações."})
}

// PATCH /api/v1/salas/:id/ativar
func (ctrl *SalaController) Ativar(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "ID inválido."})
		return
	}

	indice := BuscarIndiceSala(id, ctrl.DB.Salas)
	if indice == -1 {
		c.JSON(http.StatusNotFound, gin.H{"erro": "Sala não encontrada."})
		return
	}

	ctrl.DB.Salas[indice].Ativa = true
	c.JSON(http.StatusOK, gin.H{"mensagem": "Sala reativada com sucesso! Ela já pode receber novas alocações."})
}

// GET /api/v1/salas/:id/grade
func (ctrl *SalaController) ConsultarGrade(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "ID inválido."})
		return
	}

	indice := BuscarIndiceSala(id, ctrl.DB.Salas)
	if indice == -1 {
		c.JSON(http.StatusNotFound, gin.H{"erro": "Sala não encontrada."})
		return
	}

	sala := ctrl.DB.Salas[indice]
	var gradeDeUso []gin.H

	for _, t := range ctrl.DB.Turmas {
		if !t.Ativa {
			continue
		}
		for _, a := range t.Alocacoes {
			if a.SalaID == id {
				gradeDeUso = append(gradeDeUso, gin.H{
					"dia":            a.DiaDaSemana,
					"horario_inicio": a.HorarioInicio,
					"horario_fim":    a.HorarioFim,
					"turma":          t.Nome,
					"disciplina":     t.Disciplina,
				})
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"sala":  sala.Nome,
		"grade": gradeDeUso,
	})
}

// Funções
func CadastrarSala(id int, nome string, capacidade int, recursos []Recursos) Sala {
	if recursos == nil {
		recursos = []Recursos{}
	}
	return Sala{
		ID:         id,
		Nome:       nome,
		Capacidade: capacidade,
		Recursos:   recursos,
		Ativa:      true,
	}
}

func GerarIdSala(salas []Sala) int {
	maxID := 0
	for _, s := range salas {
		if s.ID > maxID {
			maxID = s.ID
		}
	}
	return maxID + 1
}

func BuscarIndiceSala(id int, salas []Sala) int {
	for i, sala := range salas {
		if sala.ID == id {
			return i
		}
	}
	return -1
}

func NomeDeSalaJaExiste(nome string, salas []Sala) bool {
	for _, sala := range salas {
		if strings.EqualFold(sala.Nome, nome) {
			return true
		}
	}
	return false
}

func SalaEstaAgendadaNaTurma(idSala int, alocacoes []Alocacao) bool {
	for _, aloc := range alocacoes {
		if aloc.SalaID == idSala {
			return true
		}
	}
	return false
}

func PodeReduzirCapacidadeDaSala(idSala int, novaCapacidade int, turmas []Turma) bool {
	for _, t := range turmas {
		if SalaEstaAgendadaNaTurma(idSala, t.Alocacoes) {
			if novaCapacidade < t.QntAluno {
				return false
			}
		}
	}
	return true
}

func AtualizarSalaDados(id int, novaCapacidade int, novosRecursos []Recursos, db *Database) {
	indice := BuscarIndiceSala(id, db.Salas)
	db.Salas[indice].Capacidade = novaCapacidade
	db.Salas[indice].Recursos = novosRecursos
}
