export default function Home() {
  return (
    <div className="py-8">
      <h1 className="text-4xl font-bold mb-6">UniBlack - 云黑名单系统</h1>
      <p className="text-gray-600 mb-8">
        一个可复用的通用云黑系统，支持在线提交查询、通用查询API、申诉、管理员审核和追责。
      </p>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="bg-white rounded-lg shadow-md p-6">
          <h2 className="text-xl font-semibold mb-3">查询</h2>
          <p className="text-gray-600 mb-4">通过QQ、微信、B站等平台查询某人是否在黑名单中。</p>
          <a href="/search" className="text-blue-600 hover:underline">开始查询 →</a>
        </div>

        <div className="bg-white rounded-lg shadow-md p-6">
          <h2 className="text-xl font-semibold mb-3">举报</h2>
          <p className="text-gray-600 mb-4">提交举报信息，帮助社区维护安全环境。</p>
          <a href="/submit" className="text-blue-600 hover:underline">提交举报 →</a>
        </div>

        <div className="bg-white rounded-lg shadow-md p-6">
          <h2 className="text-xl font-semibold mb-3">申诉</h2>
          <p className="text-gray-600 mb-4">如果您认为判决有误，可以提交申诉。</p>
          <a href="/login" className="text-blue-600 hover:underline">登录后申诉 →</a>
        </div>
      </div>

      <div className="mt-12">
        <h2 className="text-2xl font-bold mb-4">统计信息</h2>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="bg-blue-500 text-white rounded-lg p-4 text-center">
            <div className="text-3xl font-bold">-</div>
            <div className="text-sm">黑名单对象</div>
          </div>
          <div className="bg-green-500 text-white rounded-lg p-4 text-center">
            <div className="text-3xl font-bold">-</div>
            <div className="text-sm">已审核案件</div>
          </div>
          <div className="bg-yellow-500 text-white rounded-lg p-4 text-center">
            <div className="text-3xl font-bold">-</div>
            <div className="text-sm">待审核</div>
          </div>
          <div className="bg-purple-500 text-white rounded-lg p-4 text-center">
            <div className="text-3xl font-bold">-</div>
            <div className="text-sm">已处理申诉</div>
          </div>
        </div>
      </div>
    </div>
  )
}
