package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type Comb struct {
	ID        uint       `gorm:"primary_key" json:"id"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt"`
	Honey     []Honey    `json:"honey"`
}

type Honey struct {
	CombID    int        `gorm:"primaryKey;autoIncrement:false" json:"combId"`
	Type      string     `gorm:"primaryKey" json:"type"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt"`
	Visits    int        `json:"visits"`
}

func main() {

	ctx := context.Background()

	exporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		log.Fatalf("failed to initialize exporter: %v", err)
	}

	// Create a new tracer provider with a batch span processor and the otlp exporter.
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
	)

	// Handle shutdown errors in a sensible manner where possible
	defer func() { _ = tp.Shutdown(ctx) }()

	// Set the Tracer Provider global
	otel.SetTracerProvider(tp)

	// Register the trace context and baggage propagators so data is propagated across services/processes.
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	r := gin.Default()
	r.Use(otelgin.Middleware("combs-of-honey"))

	db, err := gorm.Open(sqlite.Open("test-comb.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	db.AutoMigrate(&Comb{})
	db.AutoMigrate(&Honey{})

	r.POST("/combs", createComb(db))
	r.GET("/combs", getCombs(db))
	r.GET("/combs/:combId", getComb(db))

	r.POST("/combs/:combId/honey", createHoney(db))
	r.GET("/combs/:combId/honey", getAllHoney(db))
	r.GET("/combs/:combId/honey/:honeyType", getHoney(db))
	r.DELETE("/combs/:combId/honey/:honeyType", deleteHoney(db))

	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}

func createComb(db *gorm.DB) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		comb := Comb{}
		result := db.Create(&comb)
		if result.Error != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		if comb.Honey == nil {
			comb.Honey = make([]Honey, 0)
		}
		c.JSON(http.StatusCreated, comb)
	}
	return gin.HandlerFunc(fn)
}

func getCombs(db *gorm.DB) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		combs := []Comb{}
		result := db.Find(&combs)

		if result.Error != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		if result.RowsAffected == 0 {
			c.Status(http.StatusNotFound)
			return
		}

		for i, comb := range combs {
			if comb.Honey == nil {
				comb.Honey = make([]Honey, 0)
			}
			combs[i] = comb
		}

		c.JSON(http.StatusOK, combs)
	}
	return gin.HandlerFunc(fn)
}

func getComb(db *gorm.DB) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		id := c.Param("combId")

		span := trace.SpanFromContext(c.Request.Context())
		span.SetAttributes(attribute.String("comb.id", id))

		comb := Comb{}
		result := db.First(&comb, id)

		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				c.Status(http.StatusNotFound)
				return
			}
			c.Status(http.StatusInternalServerError)
			return
		}

		if comb.Honey == nil {
			comb.Honey = make([]Honey, 0)
		}
		c.JSON(http.StatusOK, comb)
	}
	return gin.HandlerFunc(fn)
}

func createHoney(db *gorm.DB) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		combId, _ := strconv.Atoi(c.Param("combId"))
		honey := Honey{}

		span := trace.SpanFromContext(c.Request.Context())
		span.SetAttributes(attribute.Int("comb.id", combId))

		c.BindJSON(&honey)
		honey.CombID = combId
		result := db.Create(&honey)

		if result.Error != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		c.JSON(http.StatusCreated, honey)
	}
	return gin.HandlerFunc(fn)
}

func getAllHoney(db *gorm.DB) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		combId, _ := strconv.Atoi(c.Param("combId"))
		allHoney := []Honey{}

		span := trace.SpanFromContext(c.Request.Context())
		span.SetAttributes(attribute.Int("comb.id", combId))

		result := db.Where("comb_id = ?", combId).Find(&allHoney)

		if result.Error != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		if result.RowsAffected == 0 {
			c.Status(http.StatusNotFound)
			return
		}

		for _, honey := range allHoney {
			honey.Visits += 1
		}
		for i, honey := range allHoney {
			honey.Visits = honey.Visits + 1
			result := db.Model(&honey).Update("visits", honey.Visits)
			if result.Error != nil {
				c.Status(http.StatusInternalServerError)
				return
			}
			allHoney[i] = honey
		}
		c.JSON(http.StatusOK, allHoney)
	}
	return gin.HandlerFunc(fn)
}

func getHoney(db *gorm.DB) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		combId, _ := strconv.Atoi(c.Param("combId"))
		honeyType := c.Param("honeyType")
		honey := Honey{}

		span := trace.SpanFromContext(c.Request.Context())
		span.SetAttributes(attribute.Int("comb.id", combId))
		span.SetAttributes(attribute.String("honey.type", honeyType))

		tracer := otel.GetTracerProvider().Tracer("")
		_, span = tracer.Start(c.Request.Context(), "db-call")

		result := db.Limit(1).Where("comb_id = ? AND type = ?", combId, honeyType).Find(&honey)
		span.SetAttributes(attribute.String("db.sql", result.Statement.SQL.String()))
		span.End()

		if result.Error != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		if result.RowsAffected == 0 {
			c.Status(http.StatusNotFound)
			return
		}

		honey.Visits = honey.Visits + 1
		result = db.Model(&honey).Update("visits", honey.Visits)

		if result.Error != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		c.JSON(http.StatusOK, honey)
	}
	return gin.HandlerFunc(fn)
}

func deleteHoney(db *gorm.DB) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		combId, _ := strconv.Atoi(c.Param("combId"))
		honeyType := c.Param("honeyType")
		honey := Honey{}

		result := db.Where("comb_id = ? AND type = ?", combId, honeyType).Delete(&honey)

		if result.Error != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		c.Status(http.StatusOK)
	}
	return gin.HandlerFunc(fn)
}
