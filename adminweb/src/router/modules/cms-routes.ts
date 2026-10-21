import { AppRouteRecord } from '@/types/router'

export const cmsRoutes: AppRouteRecord = {
  path: '/cms',
  name: 'Cms',
  component: '/index/index',
  meta: {
    title: '内容管理',
    icon: 'ri:article-line',
    roles: ['R_SUPER', 'R_ADMIN']
  },
  children: [
    {
      path: 'categories',
      name: 'CmsCategories',
      component: '/cms/categories',
      meta: {
        title: '分类管理',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: '查看分类', authMark: 'cms_category.list' },
          { title: '创建分类', authMark: 'cms_category.create' },
          { title: '更新分类', authMark: 'cms_category.update' },
          { title: '删除分类', authMark: 'cms_category.delete' }
        ]
      }
    },
    {
      path: 'posts',
      name: 'CmsPosts',
      component: '/cms/posts',
      meta: {
        title: '文章管理',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: '查看文章', authMark: 'cms_post.list' },
          { title: '创建文章', authMark: 'cms_post.create' },
          { title: '更新文章', authMark: 'cms_post.update' },
          { title: '删除文章', authMark: 'cms_post.delete' }
        ]
      }
    },
    {
      path: 'blocks',
      name: 'CmsBlocks',
      component: '/cms/blocks',
      meta: {
        title: '区块管理',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: '查看区块', authMark: 'cms_block.list' },
          { title: '创建区块', authMark: 'cms_block.create' },
          { title: '更新区块', authMark: 'cms_block.update' },
          { title: '删除区块', authMark: 'cms_block.delete' }
        ]
      }
    },
    {
      path: 'uploads',
      name: 'CmsUploads',
      component: '/cms/uploads',
      meta: {
        title: '素材上传',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: '查看上传素材', authMark: 'upload.list' },
          { title: '上传素材', authMark: 'upload.create' }
        ]
      }
    },
    {
      path: 'nav-items',
      name: 'CmsNavItems',
      component: '/cms/nav-items',
      meta: {
        title: '导航配置',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: '查看导航', authMark: 'settings.view' },
          { title: '更新导航', authMark: 'settings.update' }
        ]
      }
    }
  ]
}
