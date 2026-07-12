from django.urls import path, re_path
import views

urlpatterns = [
    path("users/<int:id>", views.get_user),
    re_path(r"^admin/$", views.admin_view),
]
