package optimization

var topScores []Result

func addScore(result Result) {
	position := len(topScores)

	for i := 0; i < len(topScores); i++ {
		// If score is bigger, should replace in the position.
		if result.score > topScores[i].score {
			position = i
			break
		}

		// If equal, should replace only if kl is lower.
		if result.score == topScores[i].score && result.kl < topScores[i].kl {
			position = i
			break
		}
	}

	copy(topScores[position+1:], topScores[position:])
	topScores[position] = result
}
