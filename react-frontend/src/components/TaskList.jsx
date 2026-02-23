import { useState } from 'react'
import { getTaskHistory } from '../services/api'
import './TaskList.css'

const formatTimestamp = (timestamp) => {
  if (!timestamp) {
    return 'unknown time'
  }

  const parsed = new Date(timestamp)
  if (Number.isNaN(parsed.getTime())) {
    return timestamp
  }
  return parsed.toLocaleString()
}

const formatHistoryChange = (entry) => {
  const fromValue = entry.fromValue ?? 'empty'
  return `${entry.field}: ${fromValue} -> ${entry.toValue}`
}

function TaskList({ tasks, onTaskEdit }) {
  const [historyTask, setHistoryTask] = useState(null)
  const [historyItems, setHistoryItems] = useState([])
  const [historyLoading, setHistoryLoading] = useState(false)
  const [historyError, setHistoryError] = useState('')

  if (tasks.length === 0) {
    return <div className="empty-state">No tasks found</div>
  }

  const getStatusColor = (status) => {
    switch (status) {
      case 'completed':
        return '#4caf50'
      case 'in-progress':
        return '#ff9800'
      case 'pending':
        return '#f44336'
      default:
        return '#9e9e9e'
    }
  }

  const handleOpenHistory = async (task) => {
    setHistoryTask(task)
    setHistoryLoading(true)
    setHistoryError('')
    setHistoryItems([])

    try {
      const response = await getTaskHistory(task.id)
      setHistoryItems(response.history || [])
    } catch (error) {
      setHistoryError(error.message || 'Failed to load task history')
    } finally {
      setHistoryLoading(false)
    }
  }

  const handleCloseHistory = () => {
    setHistoryTask(null)
    setHistoryItems([])
    setHistoryLoading(false)
    setHistoryError('')
  }

  return (
    <div className="task-list">
      {tasks.map((task) => (
        <div key={task.id} className="task-card">
          <div className="task-header">
            <h3>{task.title}</h3>
            <span
              className="task-status"
              style={{ backgroundColor: getStatusColor(task.status) }}
            >
              {task.status}
            </span>
          </div>
          <div className="task-last-change">
            {task.lastChange
              ? `Last change: ${formatHistoryChange(task.lastChange)} by ${task.lastChange.changedBy} on ${formatTimestamp(task.lastChange.changedAt)}`
              : 'Last change: no changes yet'}
          </div>
          <div className="task-footer">
            <span className="task-id">Task #{task.id}</span>
            <span className="task-user">User ID: {task.userId}</span>
            <div className="task-actions">
              <button
                className="task-history-btn"
                onClick={() => handleOpenHistory(task)}
                title="View task history"
              >
                History
              </button>
              {onTaskEdit && (
                <button
                  className="task-edit-btn"
                  onClick={() => onTaskEdit(task)}
                  title="Edit task"
                >
                  Edit
                </button>
              )}
            </div>
          </div>
        </div>
      ))}

      {historyTask && (
        <div className="task-history-overlay" onClick={handleCloseHistory}>
          <div className="task-history-modal" onClick={(event) => event.stopPropagation()}>
            <div className="task-history-header">
              <h3>Task #{historyTask.id} History</h3>
              <button className="task-history-close" onClick={handleCloseHistory}>
                Close
              </button>
            </div>
            {historyLoading && <div className="task-history-state">Loading history...</div>}
            {!historyLoading && historyError && (
              <div className="task-history-error">{historyError}</div>
            )}
            {!historyLoading && !historyError && historyItems.length === 0 && (
              <div className="task-history-state">No history entries found.</div>
            )}
            {!historyLoading && !historyError && historyItems.length > 0 && (
              <ul className="task-history-list">
                {historyItems.map((entry) => (
                  <li
                    key={`${entry.id}-${entry.changedAt}-${entry.field}-${entry.toValue}`}
                    className="task-history-item"
                  >
                    <div className="task-history-change">{formatHistoryChange(entry)}</div>
                    <div className="task-history-meta">
                      <span>{formatTimestamp(entry.changedAt)}</span>
                      <span>by {entry.changedBy}</span>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

export default TaskList
