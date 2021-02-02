package queue

func (st *sqsResponseThenable) Then(callback MessageHandler) *sqsResponseThenable {
	st.queue.thens[st.messageID] = append(st.queue.thens[st.messageID], callback)
	return st
}
