from django.urls import path, re_path
import views


def get_user(request, id):
    return None


def admin_view(request):
    return None


urlpatterns = [
    path("users/<int:id>", views.get_user),
    re_path(r"^admin/$", views.admin_view),
]
