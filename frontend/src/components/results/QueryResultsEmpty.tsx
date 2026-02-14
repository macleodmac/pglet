import icon from '../../assets/icon.png'

export function QueryResultsEmpty() {
  return (
    <div className="flex h-full items-center justify-center">
      <div className="flex flex-col items-center px-6 py-8 text-center">
        <img
          src={icon}
          alt="pglet mascot"
          className="mb-2 h-20 w-20 opacity-70"
        />
        <h2 className="mb-1 text-lg font-medium text-gray-700 dark:text-gray-300">
          Ready to query
        </h2>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          Write a query and press Run to see results
        </p>
      </div>
    </div>
  )
}
