package main

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Structs
type Alocacao struct {
	SalaID        int    `json:"sala_id"`
	DiaDaSemana   string `json:"dia_da_semana"`
	HorarioInicio string `json:"horario_inicio"`
	HorarioFim    string `json:"horario_fim"`
}

type Turma struct {
	ID         int        `json:"id"`
	Nome       string     `json:"nome"`
	Disciplina string     `json:"disciplina"`
	Professor  string     `json:"professor"`
	QntAluno   int        `json:"qnt_aluno"`
	Alocacoes  []Alocacao `json:"alocacoes"`
	Alunos     []Aluno    `json:"alunos"`
	Ativa      bool       `json:"ativa"`
}

// Requests
type CriarTurmaRequest struct {
	Nome       string `json:"nome" binding:"required"`
	Disciplina string `json:"disciplina" binding:"required"`
	Professor  string `json:"professor" binding:"required"`
}

type MatricularAlunoRequest struct {
	Matricula int `json:"matricula" binding:"required,gt=0"`
}

type AlocarSalaRequest struct {
	SalaID        int    `json:"sala_id" binding:"required,gt=0"`
	DiaDaSemana   string `json:"dia_da_semana" binding:"required"`
	HorarioInicio string `json:"horario_inicio" binding:"required"`
	HorarioFim    string `json:"horario_fim" binding:"required"`
}

// Controller
type TurmaController struct {
	DB *Database
}

func NovoTurmaController(db *Database) *TurmaController {
	return &TurmaController{
		DB: db,
	}
}

// End Points
// POST /api/v1/turmas
func (ctrl *TurmaController) Criar(c *gin.Context) {
	var req CriarTurmaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "Dados inválidos: " + err.Error()})
		return
	}

	if NomeDeTurmaJaExiste(req.Nome, ctrl.DB.Turmas) {
		c.JSON(http.StatusConflict, gin.H{"erro": "Já existe uma turma com esse nome."})
		return
	}

	novoID := GerarIdTurma(ctrl.DB.Turmas)
	novaTurma := CadastrarTurma(novoID, req.Nome, req.Disciplina, req.Professor)

	ctrl.DB.Turmas = append(ctrl.DB.Turmas, novaTurma)

	c.JSON(http.StatusCreated, gin.H{
		"mensagem": "Turma cadastrada com sucesso!",
		"turma":    novaTurma,
	})
}

// GET /api/v1/turmas
func (ctrl *TurmaController) Listar(c *gin.Context) {
	if len(ctrl.DB.Turmas) == 0 {
		c.JSON(http.StatusOK, gin.H{"mensagem": "Nenhuma turma cadastrada.", "turmas": []gin.H{}})
		return
	}

	var turmasResponse []gin.H
	for _, t := range ctrl.DB.Turmas {
		statusAlocacao := "NAO_ALOCADA"
		if len(t.Alocacoes) > 0 {
			statusAlocacao = "ALOCADA"
		}

		turmasResponse = append(turmasResponse, gin.H{
			"id":              t.ID,
			"nome":            t.Nome,
			"disciplina":      t.Disciplina,
			"professor":       t.Professor,
			"qnt_aluno":       t.QntAluno,
			"alocacoes":       t.Alocacoes,
			"alunos":          t.Alunos,
			"ativa":           t.Ativa,
			"status_alocacao": statusAlocacao,
		})
	}

	c.JSON(http.StatusOK, gin.H{"turmas": turmasResponse})
}

// PATCH /api/v1/turmas/:id/inativar
func (ctrl *TurmaController) Inativar(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "ID inválido."})
		return
	}

	indice := BuscarIndiceTurma(id, ctrl.DB.Turmas)
	if indice == -1 {
		c.JSON(http.StatusNotFound, gin.H{"erro": "Turma não encontrada."})
		return
	}

	ctrl.DB.Turmas[indice].Ativa = false
	c.JSON(http.StatusOK, gin.H{"mensagem": "Turma inativada com sucesso. (Ela não gerará mais conflitos de horário)"})
}

// POST /api/v1/turmas/:id/alunos
func (ctrl *TurmaController) MatricularAluno(c *gin.Context) {
	idTurma, err := strconv.Atoi(c.Param("id"))
	if err != nil || idTurma <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "ID da turma inválido."})
		return
	}

	var req MatricularAlunoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "Dados inválidos."})
		return
	}

	idxTurma := BuscarIndiceTurma(idTurma, ctrl.DB.Turmas)
	if idxTurma == -1 {
		c.JSON(http.StatusNotFound, gin.H{"erro": "Turma não encontrada."})
		return
	}

	turma := ctrl.DB.Turmas[idxTurma]
	if !turma.Ativa {
		c.JSON(http.StatusForbidden, gin.H{"erro": "Esta turma está inativ, não é possível alterar alunos."})
		return
	}

	idxAluno := BuscarIndiceAluno(req.Matricula, ctrl.DB.Alunos)
	if idxAluno == -1 {
		c.JSON(http.StatusNotFound, gin.H{"erro": "Este aluno não existe no sistema."})
		return
	}

	if AlunoJaNaTurma(req.Matricula, turma.Alunos) {
		c.JSON(http.StatusConflict, gin.H{"erro": "Este aluno já está matriculado nesta turma."})
		return
	}

	if !PodeAdicionarAluno(turma, ctrl.DB.Salas) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": "A turma já atingiu o limite de capacidade em uma das salas agendadas."})
		return
	}

	if AlunoTemConflitoDeHorario(req.Matricula, turma, ctrl.DB.Turmas) {
		c.JSON(http.StatusConflict, gin.H{"erro": "Conflito: Este aluno já possui aula em outra turma neste mesmo horário!"})
		return
	}

	ctrl.DB.Turmas[idxTurma].Alunos = append(ctrl.DB.Turmas[idxTurma].Alunos, ctrl.DB.Alunos[idxAluno])
	ctrl.DB.Turmas[idxTurma].QntAluno++

	c.JSON(http.StatusOK, gin.H{"mensagem": "Aluno matriculado com sucesso!"})
}

// GET /api/v1/turmas/:id/alunos
func (ctrl *TurmaController) ListarAlunos(c *gin.Context) {
	idTurma, err := strconv.Atoi(c.Param("id"))
	if err != nil || idTurma <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "ID da turma inválido."})
		return
	}

	idxTurma := BuscarIndiceTurma(idTurma, ctrl.DB.Turmas)
	if idxTurma == -1 {
		c.JSON(http.StatusNotFound, gin.H{"erro": "Turma não encontrada."})
		return
	}

	alunos := ctrl.DB.Turmas[idxTurma].Alunos
	if len(alunos) == 0 {
		c.JSON(http.StatusOK, gin.H{"mensagem": "Nenhum aluno matriculado nesta turma.", "alunos": []Aluno{}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"alunos": alunos})
}

// DELETE /api/v1/turmas/:id/alunos/:matricula
func (ctrl *TurmaController) RemoverAluno(c *gin.Context) {
	idTurma, errT := strconv.Atoi(c.Param("id"))
	matricula, errM := strconv.Atoi(c.Param("matricula"))

	if errT != nil || errM != nil || idTurma <= 0 || matricula <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "ID da turma ou matrícula inválidos."})
		return
	}

	idxTurma := BuscarIndiceTurma(idTurma, ctrl.DB.Turmas)
	if idxTurma == -1 {
		c.JSON(http.StatusNotFound, gin.H{"erro": "Turma não encontrada."})
		return
	}

	turma := ctrl.DB.Turmas[idxTurma]

	if !turma.Ativa {
		c.JSON(http.StatusForbidden, gin.H{"erro": "Esta turma está inativa. Não é possível remover alunos."})
		return
	}

	idxAluno := BuscarIndiceAluno(matricula, turma.Alunos)
	if idxAluno == -1 {
		c.JSON(http.StatusNotFound, gin.H{"erro": "Aluno não está nesta turma."})
		return
	}

	ctrl.DB.Turmas[idxTurma].Alunos = append(turma.Alunos[:idxAluno], turma.Alunos[idxAluno+1:]...)
	ctrl.DB.Turmas[idxTurma].QntAluno--

	c.JSON(http.StatusOK, gin.H{"mensagem": "Aluno removido da turma com sucesso."})
}

// POST /api/v1/turmas/:id/alocacoes
func (ctrl *TurmaController) AlocarSala(c *gin.Context) {
	idTurma, err := strconv.Atoi(c.Param("id"))
	if err != nil || idTurma <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "ID da turma inválido."})
		return
	}

	var req AlocarSalaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "Dados inválidos: " + err.Error()})
		return
	}

	idxTurma := BuscarIndiceTurma(idTurma, ctrl.DB.Turmas)
	if idxTurma == -1 {
		c.JSON(http.StatusNotFound, gin.H{"erro": "Turma não encontrada."})
		return
	}
	turma := ctrl.DB.Turmas[idxTurma]
	if !turma.Ativa {
		c.JSON(http.StatusForbidden, gin.H{"erro": "Turma inativa não recebe alocações."})
		return
	}

	idxSala := BuscarIndiceSala(req.SalaID, ctrl.DB.Salas)
	if idxSala == -1 {
		c.JSON(http.StatusNotFound, gin.H{"erro": "Sala não encontrada."})
		return
	}
	sala := ctrl.DB.Salas[idxSala]

	if !sala.Ativa {
		c.JSON(http.StatusForbidden, gin.H{"erro": "A sala informada está inativa."})
		return
	}
	if sala.Capacidade < turma.QntAluno {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": "A sala é menor que a quantidade de alunos já matriculados."})
		return
	}

	if !ValidarHorario(req.HorarioInicio) || !ValidarHorario(req.HorarioFim) || req.HorarioInicio >= req.HorarioFim {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "Horários inválidos ou horário de fim é menor que o de início."})
		return
	}

	if SalaEstaOcupadaNesteHorario(sala.ID, req.DiaDaSemana, req.HorarioInicio, req.HorarioFim, ctrl.DB.Turmas) {
		c.JSON(http.StatusConflict, gin.H{"erro": "Esta sala já está reservada para outra turma neste horário."})
		return
	}

	if NovoHorarioCausaConflitoAosAlunos(req.DiaDaSemana, req.HorarioInicio, req.HorarioFim, turma, ctrl.DB.Turmas) {
		c.JSON(http.StatusConflict, gin.H{"erro": "conflito na grade dos alunos já matriculados."})
		return
	}

	novaAlocacao := Alocacao{
		SalaID:        sala.ID,
		DiaDaSemana:   req.DiaDaSemana,
		HorarioInicio: req.HorarioInicio,
		HorarioFim:    req.HorarioFim,
	}

	ctrl.DB.Turmas[idxTurma].Alocacoes = append(ctrl.DB.Turmas[idxTurma].Alocacoes, novaAlocacao)
	c.JSON(http.StatusCreated, gin.H{"mensagem": "Sala alocada com sucesso!"})
}

// DELETE /api/v1/turmas/:id/alocacoes/:indexAlocacao
func (ctrl *TurmaController) RemoverAlocacao(c *gin.Context) {
	idTurma, errT := strconv.Atoi(c.Param("id"))
	idxAlocacao, errA := strconv.Atoi(c.Param("indexAlocacao"))

	if errT != nil || errA != nil || idTurma <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "ID da turma ou índice de alocação inválidos."})
		return
	}

	idxTurma := BuscarIndiceTurma(idTurma, ctrl.DB.Turmas)
	if idxTurma == -1 {
		c.JSON(http.StatusNotFound, gin.H{"erro": "Turma não encontrada."})
		return
	}

	turma := ctrl.DB.Turmas[idxTurma]

	if !turma.Ativa {
		c.JSON(http.StatusForbidden, gin.H{"erro": "Esta turma está inativa. Não é possível alterar alocações."})
		return
	}

	if idxAlocacao < 0 || idxAlocacao >= len(turma.Alocacoes) {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "Índice de alocação inválido."})
		return
	}

	ctrl.DB.Turmas[idxTurma].Alocacoes = append(turma.Alocacoes[:idxAlocacao], turma.Alocacoes[idxAlocacao+1:]...)
	c.JSON(http.StatusOK, gin.H{"mensagem": "Alocação removida com sucesso."})
}

// Func
func CadastrarTurma(id int, nome string, disciplina string, professor string) Turma {
	return Turma{
		ID:         id,
		Nome:       nome,
		Disciplina: disciplina,
		Professor:  professor,
		QntAluno:   0,
		Alocacoes:  []Alocacao{},
		Alunos:     []Aluno{},
		Ativa:      true,
	}
}

func ExisteConflitoDeHorario(dia1, inicio1, fim1, dia2, inicio2, fim2 string) bool {
	if !strings.EqualFold(dia1, dia2) {
		return false
	}
	return inicio1 < fim2 && fim1 > inicio2
}

func SalaEstaOcupadaNesteHorario(idSala int, dia, inicio, fim string, turmas []Turma) bool {
	for _, t := range turmas {
		if !t.Ativa {
			continue
		}
		for _, aloc := range t.Alocacoes {
			if aloc.SalaID == idSala {
				if ExisteConflitoDeHorario(dia, inicio, fim, aloc.DiaDaSemana, aloc.HorarioInicio, aloc.HorarioFim) {
					return true
				}
			}
		}
	}
	return false
}

func AlunoTemConflitoDeHorario(matricula int, novaTurma Turma, turmas []Turma) bool {
	for _, turmaExistente := range turmas {
		if !turmaExistente.Ativa || turmaExistente.ID == novaTurma.ID {
			continue
		}
		if AlunoJaNaTurma(matricula, turmaExistente.Alunos) {
			for _, horarioNovo := range novaTurma.Alocacoes {
				for _, horarioAntigo := range turmaExistente.Alocacoes {
					if ExisteConflitoDeHorario(horarioNovo.DiaDaSemana, horarioNovo.HorarioInicio, horarioNovo.HorarioFim,
						horarioAntigo.DiaDaSemana, horarioAntigo.HorarioInicio, horarioAntigo.HorarioFim) {
						return true
					}
				}
			}
		}
	}
	return false
}

func NovoHorarioCausaConflitoAosAlunos(dia, inicio, fim string, turmaAtual Turma, turmas []Turma) bool {
	for _, aluno := range turmaAtual.Alunos {
		for _, outraTurma := range turmas {
			if !outraTurma.Ativa || outraTurma.ID == turmaAtual.ID {
				continue
			}
			if AlunoJaNaTurma(aluno.Matricula, outraTurma.Alunos) {
				for _, horarioOutraTurma := range outraTurma.Alocacoes {
					if ExisteConflitoDeHorario(dia, inicio, fim, horarioOutraTurma.DiaDaSemana, horarioOutraTurma.HorarioInicio, horarioOutraTurma.HorarioFim) {
						return true
					}
				}
			}
		}
	}
	return false
}

func GerarIdTurma(turmas []Turma) int {
	maxID := 0
	for _, t := range turmas {
		if t.ID > maxID {
			maxID = t.ID
		}
	}
	return maxID + 1
}

func BuscarIndiceTurma(id int, turmas []Turma) int {
	for i, t := range turmas {
		if t.ID == id {
			return i
		}
	}
	return -1
}

func NomeDeTurmaJaExiste(nome string, turmas []Turma) bool {
	for _, t := range turmas {
		if strings.EqualFold(t.Nome, nome) {
			return true
		}
	}
	return false
}

func AlunoJaNaTurma(matricula int, alunosDaTurma []Aluno) bool {
	for _, a := range alunosDaTurma {
		if a.Matricula == matricula {
			return true
		}
	}
	return false
}

func PodeAdicionarAluno(turma Turma, salas []Sala) bool {
	novoTotal := turma.QntAluno + 1
	for _, aloc := range turma.Alocacoes {
		idxSala := BuscarIndiceSala(aloc.SalaID, salas)
		if idxSala != -1 {
			if novoTotal > salas[idxSala].Capacidade {
				return false
			}
		} else {
			return false
		}
	}
	return true
}
