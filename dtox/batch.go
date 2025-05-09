package dtox

import (
	"fmt"
	"sync"
)

// ToModels converts a slice of DTOs to a slice of models
func (m *Mapper[TDto, TModel]) ToModels(dtos []TDto) ([]TModel, error) {
	if len(dtos) == 0 {
		return []TModel{}, nil
	}

	// Check if parallel processing is enabled
	if m.options.UseParallelProcessing && len(dtos) >= m.options.ParallelThreshold {
		return m.toModelsParallel(dtos)
	}

	// Sequential processing
	models := make([]TModel, 0, len(dtos))

	for _, dto := range dtos {
		model, err := m.ToModel(dto)
		if err != nil {
			return nil, err
		}
		models = append(models, model)
	}

	return models, nil
}

// ToDtos converts a slice of models to a slice of DTOs
func (m *Mapper[TDto, TModel]) ToDtos(models []TModel) ([]TDto, error) {
	if len(models) == 0 {
		return []TDto{}, nil
	}

	// Check if parallel processing is enabled
	if m.options.UseParallelProcessing && len(models) >= m.options.ParallelThreshold {
		return m.toDtosParallel(models)
	}

	// Sequential processing
	dtos := make([]TDto, 0, len(models))

	for _, model := range models {
		dto, err := m.ToDto(model)
		if err != nil {
			return nil, err
		}
		dtos = append(dtos, dto)
	}

	return dtos, nil
}

// toModelsParallel converts DTOs to models in parallel using goroutines
func (m *Mapper[TDto, TModel]) toModelsParallel(dtos []TDto) ([]TModel, error) {
	models := make([]TModel, len(dtos))
	errChan := make(chan error, 1)
	var wg sync.WaitGroup
	var once sync.Once

	// Process each DTO in its own goroutine
	for i, dto := range dtos {
		wg.Add(1)
		go func(index int, item TDto) {
			defer wg.Done()

			model, err := m.ToModel(item)
			if err != nil {
				// Report only the first error
				once.Do(func() {
					// Create a batch error with details
					batchErr := ErrorRegistry.NewWithCause(ErrBatchConversion, err).
						WithDetail("index", index).
						WithDetail("original_error", err.Error())
					errChan <- batchErr
				})
				return
			}

			models[index] = model
		}(i, dto)
	}

	// Wait for all conversions to complete or for an error
	wg.Wait()
	close(errChan)

	// Check if there was an error
	if err := <-errChan; err != nil {
		return nil, err
	}

	return models, nil
}

// toDtosParallel converts models to DTOs in parallel using goroutines
func (m *Mapper[TDto, TModel]) toDtosParallel(models []TModel) ([]TDto, error) {
	dtos := make([]TDto, len(models))
	errChan := make(chan error, 1)
	var wg sync.WaitGroup
	var once sync.Once

	// Process each model in its own goroutine
	for i, model := range models {
		wg.Add(1)
		go func(index int, item TModel) {
			defer wg.Done()

			dto, err := m.ToDto(item)
			if err != nil {
				// Report only the first error
				once.Do(func() {
					errChan <- fmt.Errorf("error converting item at index %d: %w", index, err)
				})
				return
			}

			dtos[index] = dto
		}(i, model)
	}

	// Wait for all conversions to complete or for an error
	wg.Wait()
	close(errChan)

	// Check if there was an error
	if err := <-errChan; err != nil {
		return nil, err
	}

	return dtos, nil
}
