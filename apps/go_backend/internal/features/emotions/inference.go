package emotions

import (
	"math"
)

// =============================================================================
// INTERNAL TYPES
// =============================================================================

// emotionInput represents a single emotion input for inference calculation.
type emotionInput struct {
	emotion    *Emotion
	intensity  float64
	isNegative bool
}

// =============================================================================
// INFERRED EMOTION CALCULATION
// =============================================================================

// InferFromItems calculates the inferred emotion from task items.
// This is computed once at write time (create/update), not on read.
//
// Algorithm:
//  1. Extract all items that have emotion_id
//  2. Calculate weighted centroid using emotion's default Intensity as weight
//  3. Determine quadrant from centroid position
//  4. Find closest matching emotion from grid
//  5. Calculate dissonance score (internal conflict)
func InferFromItems(positives, negatives []TaskItem) *InferredEmotion {
	// Collect all items with emotions
	var inputs []emotionInput
	positiveCount := 0
	negativeCount := 0

	// Process positives
	for _, item := range positives {
		if item.EmotionID == nil || *item.EmotionID == "" {
			continue
		}
		emotion := GetByID(*item.EmotionID)
		if emotion == nil {
			continue
		}
		// Use emotion's default Intensity as weight
		inputs = append(inputs, emotionInput{emotion: emotion, intensity: emotion.Intensity, isNegative: false})
		positiveCount++
	}

	// Process negatives
	for _, item := range negatives {
		if item.EmotionID == nil || *item.EmotionID == "" {
			continue
		}
		emotion := GetByID(*item.EmotionID)
		if emotion == nil {
			continue
		}
		// Use emotion's default Intensity as weight
		inputs = append(inputs, emotionInput{emotion: emotion, intensity: emotion.Intensity, isNegative: true})
		negativeCount++
	}

	// If no emotions, return nil (no inferred emotion)
	if len(inputs) == 0 {
		return nil
	}

	// Calculate weighted centroid
	var totalWeight float64
	var sumV, sumA, sumD float64

	for _, inp := range inputs {
		totalWeight += inp.intensity
		sumV += inp.emotion.Valence * inp.intensity
		sumA += inp.emotion.Arousal * inp.intensity
		sumD += inp.emotion.Dominance * inp.intensity
	}

	centroidV := sumV / totalWeight
	centroidA := sumA / totalWeight
	centroidD := sumD / totalWeight

	// Determine quadrant from centroid
	quadrant := determineQuadrant(centroidV, centroidA)

	// Find closest emotion
	closestID := findClosestEmotion(centroidV, centroidA, centroidD)

	// Calculate dissonance (conflict between positive and negative valence)
	dissonance := calculateDissonance(inputs)

	return &InferredEmotion{
		Valence:          round3(centroidV),
		Arousal:          round3(centroidA),
		Dominance:        round3(centroidD),
		Quadrant:         quadrant,
		ClosestEmotionID: closestID,
		PositiveCount:    positiveCount,
		NegativeCount:    negativeCount,
		Dissonance:       round3(dissonance),
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// determineQuadrant returns the quadrant based on Valence and Arousal.
func determineQuadrant(valence, arousal float64) string {
	if valence >= 0 && arousal >= 0 {
		return "yellow" // High Energy + Pleasant
	}
	if valence >= 0 && arousal < 0 {
		return "green" // Low Energy + Pleasant
	}
	if valence < 0 && arousal >= 0 {
		return "red" // High Energy + Unpleasant
	}
	return "blue" // Low Energy + Unpleasant
}

// findClosestEmotion finds the emotion closest to the given coordinates.
// Uses 3D Euclidean distance (Valence, Arousal, Dominance).
func findClosestEmotion(valence, arousal, dominance float64) string {
	var closestDist float64 = math.MaxFloat64
	var closestEmotion *Emotion

	allEmotions := GetAll()
	for _, emotion := range allEmotions {
		dist := math.Sqrt(
			math.Pow(emotion.Valence-valence, 2) +
				math.Pow(emotion.Arousal-arousal, 2) +
				math.Pow(emotion.Dominance-dominance, 2),
		)
		if dist < closestDist {
			closestDist = dist
			closestEmotion = emotion
		}
	}

	if closestEmotion != nil {
		return closestEmotion.ID
	}
	return ""
}

// calculateDissonance measures internal conflict when positive and negative
// emotions are present together.
//
// Returns 0-1 where:
//   - 0 = No dissonance (all emotions are aligned)
//   - 1 = Maximum dissonance (opposing emotions)
func calculateDissonance(inputs []emotionInput) float64 {
	var positiveInputs, negativeInputs []emotionInput

	for _, inp := range inputs {
		if inp.emotion.Valence > 0.2 {
			positiveInputs = append(positiveInputs, inp)
		} else if inp.emotion.Valence < -0.2 {
			negativeInputs = append(negativeInputs, inp)
		}
	}

	// No dissonance if all emotions are in one valence group
	if len(positiveInputs) == 0 || len(negativeInputs) == 0 {
		return 0
	}

	// Calculate centroids for each group
	posCentroid := centroid(positiveInputs)
	negCentroid := centroid(negativeInputs)

	// Calculate angle between centroids (normalized to 0-1)
	dotProd := posCentroid[0]*negCentroid[0] + posCentroid[1]*negCentroid[1] + posCentroid[2]*negCentroid[2]
	magPos := math.Sqrt(posCentroid[0]*posCentroid[0] + posCentroid[1]*posCentroid[1] + posCentroid[2]*posCentroid[2])
	magNeg := math.Sqrt(negCentroid[0]*negCentroid[0] + negCentroid[1]*negCentroid[1] + negCentroid[2]*negCentroid[2])

	if magPos == 0 || magNeg == 0 {
		return 0
	}

	cosAngle := dotProd / (magPos * magNeg)
	// Clamp to [-1, 1] to handle floating point errors
	cosAngle = math.Max(-1, math.Min(1, cosAngle))
	angle := math.Acos(cosAngle)

	// Normalize to 0-1 (π radians = max dissonance)
	return angle / math.Pi
}

// centroid calculates the weighted center point for a set of emotion inputs.
func centroid(inputs []emotionInput) [3]float64 {
	if len(inputs) == 0 {
		return [3]float64{0, 0, 0}
	}

	var totalWeight float64
	var sumV, sumA, sumD float64

	for _, inp := range inputs {
		totalWeight += inp.intensity
		sumV += inp.emotion.Valence * inp.intensity
		sumA += inp.emotion.Arousal * inp.intensity
		sumD += inp.emotion.Dominance * inp.intensity
	}

	return [3]float64{
		sumV / totalWeight,
		sumA / totalWeight,
		sumD / totalWeight,
	}
}

// round3 rounds to 3 decimal places.
func round3(f float64) float64 {
	return math.Round(f*1000) / 1000
}

// =============================================================================
// SINGLE EMOTION INFERENCE
// =============================================================================

// InferFromSingle creates an InferredEmotion from a single emotion selection.
// Used when task has only emotion_id (no positives/negatives with emotions).
// Uses the emotion's default Intensity for weighting.
func InferFromSingle(emotionID string) *InferredEmotion {
	emotion := GetByID(emotionID)
	if emotion == nil {
		return nil
	}

	// Use emotion's default intensity as weight
	weight := emotion.Intensity

	return &InferredEmotion{
		Valence:          round3(emotion.Valence * weight),
		Arousal:          round3(emotion.Arousal * weight),
		Dominance:        round3(emotion.Dominance * weight),
		Quadrant:         emotion.Quadrant,
		ClosestEmotionID: emotion.ID,
		PositiveCount:    0,
		NegativeCount:    0,
		Dissonance:       0,
	}
}
